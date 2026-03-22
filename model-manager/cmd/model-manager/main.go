package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/simanadler/llm-d-edge/model-manager/pkg/converter"
	"github.com/simanadler/llm-d-edge/model-manager/pkg/config"
	"github.com/simanadler/llm-d-edge/model-manager/pkg/downloader"
	"github.com/simanadler/llm-d-edge/model-manager/pkg/installer"
	"github.com/simanadler/llm-d-edge/model-manager/pkg/recommender"
	"github.com/simanadler/llm-d-edge/model-manager/pkg/types"
	"github.com/spf13/cobra"
)

var (
	version = "0.1.0"
	
	// CLI flags for recommendations (all optional with sensible defaults)
	tasks              []string
	qualityPref        string
	privacyReq         string
	storageLimitGB     int
	latencyToleranceMS int
	responsiveness     string // User-friendly: "interactive", "balanced", "batch"
	
	// CLI flags for install command
	installFormat          string
	installQuantization    string
	installModelsDir       string
	installPriority        int
	installedModelsYAMLPath string
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "model-manager",
	Short: "Advanced Model Manager for llm-d-edge",
	Long: `Advanced Model Manager intelligently recommends and manages 
local LLM models based on device capabilities and user needs.`,
	Version: version,
}

var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Profile the current device",
	Long:  `Analyze device hardware and generate a capability profile.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		
		engine, err := recommender.NewEngine("")
		if err != nil {
			return fmt.Errorf("failed to create engine: %w", err)
		}

		profile, err := engine.ProfileDevice(ctx)
		if err != nil {
			return fmt.Errorf("failed to profile device: %w", err)
		}

		// Output as JSON
		output, err := json.MarshalIndent(profile, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal profile: %w", err)
		}

		fmt.Println(string(output))
		return nil
	},
}

var recommendCmd = &cobra.Command{
	Use:   "recommend",
	Short: "Get model recommendations",
	Long: `Analyze device and recommend suitable models based on your tasks and preferences.

All flags are optional. Without flags, uses sensible defaults for general use.

Examples:
  # General recommendations (default)
  model-manager recommend

  # Specify tasks
  model-manager recommend --tasks code,reasoning,chat

  # High quality preference
  model-manager recommend --tasks code --quality high

  # Interactive use (fast response needed)
  model-manager recommend --quality high --responsiveness interactive

  # Batch processing (quality over speed)
  model-manager recommend --quality high --responsiveness batch

  # All options
  model-manager recommend --tasks code,chat --quality high --responsiveness balanced`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		
		engine, err := recommender.NewEngine("")
		if err != nil {
			return fmt.Errorf("failed to create engine: %w", err)
		}

		// Build user needs from flags (uses defaults if not specified)
		needs := buildUserNeeds(tasks, qualityPref, privacyReq, storageLimitGB, latencyToleranceMS, responsiveness)

		recommendations, err := engine.GenerateRecommendations(ctx, needs)
		if err != nil {
			return fmt.Errorf("failed to generate recommendations: %w", err)
		}

		// Display recommendations
		fmt.Println("Model Recommendations")
		fmt.Println("====================")
		if len(tasks) > 0 {
			fmt.Printf("Tasks: %s\n", strings.Join(tasks, ", "))
		}
		if qualityPref != "medium" {
			fmt.Printf("Quality Preference: %s\n", qualityPref)
		}
		fmt.Println()

		for _, rec := range recommendations {
			fmt.Printf("Rank %d: %s (%s)\n", rec.Rank, rec.Model.Name, rec.Model.ParameterCount)
			fmt.Printf("  Score: %.2f\n", rec.Score)
			fmt.Printf("  Summary: %s\n", rec.Explanation.Summary)
			fmt.Printf("  Format: %s, Quantization: %s\n", 
				rec.Setup.InstallCommand, rec.Setup.EstimatedTime)
			
			if len(rec.Explanation.Strengths) > 0 {
				fmt.Printf("  Strengths:\n")
				for _, s := range rec.Explanation.Strengths {
					fmt.Printf("    - %s\n", s)
				}
			}
			
			if len(rec.Explanation.Tradeoffs) > 0 {
				fmt.Printf("  Tradeoffs:\n")
				for _, t := range rec.Explanation.Tradeoffs {
					fmt.Printf("    - %s\n", t)
				}
			}
			
			fmt.Println()
		}

		return nil
	},
}

var recommendJSONCmd = &cobra.Command{
	Use:   "recommend-json",
	Short: "Get model recommendations as JSON",
	Long:  `Analyze device and recommend suitable models, output as JSON.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		
		engine, err := recommender.NewEngine("")
		if err != nil {
			return fmt.Errorf("failed to create engine: %w", err)
		}

		// Build user needs from flags (uses defaults if not specified)
		needs := buildUserNeeds(tasks, qualityPref, privacyReq, storageLimitGB, latencyToleranceMS, responsiveness)

		recommendations, err := engine.GenerateRecommendations(ctx, needs)
		if err != nil {
			return fmt.Errorf("failed to generate recommendations: %w", err)
		}

		// Output as JSON
		output, err := json.MarshalIndent(recommendations, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal recommendations: %w", err)
		}

		fmt.Println(string(output))
		return nil
	},
}

var selectCmd = &cobra.Command{
	Use:   "select",
	Short: "Interactively select and install models",
	Long: `Analyze device, show recommendations, and interactively select models to install.
This command will:
1. Profile your device
2. Generate model recommendations
3. Let you select one or more models
4. Download and install selected models
5. Generate YAML configuration for installed models`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		
		engine, err := recommender.NewEngine("")
		if err != nil {
			return fmt.Errorf("failed to create engine: %w", err)
		}

		// Build user needs from flags
		needs := buildUserNeeds(tasks, qualityPref, privacyReq, storageLimitGB, latencyToleranceMS, responsiveness)

		// Generate recommendations
		fmt.Println("Analyzing your device and generating recommendations...")
		recommendations, err := engine.GenerateRecommendations(ctx, needs)
		if err != nil {
			return fmt.Errorf("failed to generate recommendations: %w", err)
		}

		// Display recommendations
		fmt.Println("\n" + strings.Repeat("=", 70))
		fmt.Println("Model Recommendations")
		fmt.Println(strings.Repeat("=", 70))
		
		for i, rec := range recommendations {
			fmt.Printf("\n[%d] %s (%s)\n", i+1, rec.Model.Name, rec.Model.ParameterCount)
			fmt.Printf("    Score: %.2f | %s\n", rec.Score, rec.Explanation.Summary)
			fmt.Printf("    Size: %.1f GB | Format: %s\n", 
				rec.Model.DownloadSizeGB, strings.Join(rec.Model.Formats, ", "))
			if len(rec.Explanation.Strengths) > 0 {
				fmt.Printf("    Strengths: %s\n", rec.Explanation.Strengths[0])
			}
		}

		// Prompt for selection
		fmt.Println("\n" + strings.Repeat("=", 70))
		fmt.Print("Select models to install (comma-separated numbers, e.g., 1,3): ")
		
		var input string
		fmt.Scanln(&input)
		
		if input == "" {
			fmt.Println("No models selected. Exiting.")
			return nil
		}

		// Parse selections
		selections := strings.Split(input, ",")
		selectedModels := make([]types.ModelRecommendation, 0)
		
		for _, sel := range selections {
			sel = strings.TrimSpace(sel)
			var idx int
			if _, err := fmt.Sscanf(sel, "%d", &idx); err != nil {
				fmt.Printf("Invalid selection: %s\n", sel)
				continue
			}
			
			if idx < 1 || idx > len(recommendations) {
				fmt.Printf("Invalid selection: %d (out of range)\n", idx)
				continue
			}
			
			selectedModels = append(selectedModels, recommendations[idx-1])
		}

		if len(selectedModels) == 0 {
			fmt.Println("No valid models selected. Exiting.")
			return nil
		}

		// Install selected models
		fmt.Printf("\nInstalling %d model(s)...\n", len(selectedModels))
		
		modelInstaller := createModelInstaller(installModelsDir, &recommendations[0].Model)
		
		for i, rec := range selectedModels {
			fmt.Printf("\n[%d/%d] Installing %s...\n", i+1, len(selectedModels), rec.Model.Name)
			
			// Determine format and quantization
			format := installFormat
			if format == "" {
				if len(rec.Model.Formats) > 0 {
					format = rec.Model.Formats[0]
				} else {
					format = "mlx"
				}
			}
			
			quant := installQuantization
			if quant == "" {
				quant = rec.Setup.InstallCommand // Use recommended quantization
				if quant == "" {
					quant = "q4"
				}
			}
			
			// Install model
			opts := installer.InstallOptions{
				Format:       format,
				Quantization: quant,
				ProgressCallback: func(progress downloader.DownloadProgress) {
					if progress.Percentage > 0 {
						fmt.Printf("\r    Progress: %.1f%% (%s)", progress.Percentage, progress.Status)
					}
				},
			}
			
			result, err := modelInstaller.Install(ctx, rec.Model, opts)
			if err != nil {
				fmt.Printf("\n    Error: %v\n", err)
				continue
			}
			
			fmt.Printf("\n    Installed to: %s\n", result.InstallPath)
		}

		// Generate YAML configuration for all installed models
		fmt.Println("\nGenerating YAML configuration for installed models...")
		yamlGen := config.NewYAMLGenerator(installModelsDir, installedModelsYAMLPath)
		if err := yamlGen.GenerateYAML(); err != nil {
			fmt.Printf("Warning: Failed to generate YAML: %v\n", err)
		} else {
			fmt.Printf("Generated configuration: %s\n", yamlGen.GetOutputPath())
			fmt.Println("Copy the models you want into your edge-router config.yaml")
		}

		fmt.Println("\n" + strings.Repeat("=", 70))
		fmt.Println("Installation complete!")
		fmt.Printf("Installed %d model(s) successfully.\n", len(selectedModels))
		
		return nil
	},
}

var installCmd = &cobra.Command{
	Use:   "install [model-name]",
	Short: "Install a specific model",
	Long: `Download and install a specific model by name.
The model will be downloaded, installed, and a YAML configuration will be generated.

Examples:
  model-manager install Qwen/Qwen2.5-3B-Instruct
  model-manager install Qwen/Qwen2.5-3B-Instruct --format mlx --quantization q8`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		modelName := args[0]
		
		fmt.Printf("Installing model: %s\n", modelName)
		
		// Create a basic model metadata
		// In a real implementation, we would fetch this from a registry
		model := types.ModelMetadata{
			Name:            modelName,
			HuggingFaceRepo: modelName,
			ParameterCount:  "3B", // Default, should be detected
			Formats:         []string{"mlx", "gguf"},
			Quantizations:   []string{"q4", "q8"},
		}
		
		// Determine format and quantization
		format := installFormat
		if format == "" {
			format = "mlx"
		}
		
		quant := installQuantization
		if quant == "" {
			quant = "q4"
		}
		
		// Install model
		modelInstaller := createModelInstaller(installModelsDir, &model)
		
		opts := installer.InstallOptions{
			Format:       format,
			Quantization: quant,
			ProgressCallback: func(progress downloader.DownloadProgress) {
				if progress.Percentage > 0 {
					fmt.Printf("\rProgress: %.1f%% (%s)", progress.Percentage, progress.Status)
				}
			},
		}
		
		result, err := modelInstaller.Install(ctx, model, opts)
		if err != nil {
			return fmt.Errorf("failed to install model: %w", err)
		}
		
		fmt.Printf("\nInstalled to: %s\n", result.InstallPath)
		
		// Generate YAML configuration for all installed models
		fmt.Println("\nGenerating YAML configuration for installed models...")
		yamlGen := config.NewYAMLGenerator(installModelsDir, installedModelsYAMLPath)
		if err := yamlGen.GenerateYAML(); err != nil {
			fmt.Printf("Warning: Failed to generate YAML: %v\n", err)
		} else {
			fmt.Printf("Generated configuration: %s\n", yamlGen.GetOutputPath())
			fmt.Println("Copy the models you want into your edge-router config.yaml")
		}
		
		fmt.Println("\nInstallation complete!")
		return nil
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed models",
	Long:  `List all models that have been installed on this device.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		modelInstaller := installer.NewModelInstaller(installModelsDir)
		
		installed, err := modelInstaller.ListInstalled()
		if err != nil {
			return fmt.Errorf("failed to list installed models: %w", err)
		}
		
		if len(installed) == 0 {
			fmt.Println("No models installed.")
			return nil
		}
		
		fmt.Println("Installed Models")
		fmt.Println(strings.Repeat("=", 70))
		
		for _, model := range installed {
			fmt.Printf("\n%s\n", model.Model.Name)
			fmt.Printf("  Path: %s\n", model.InstallPath)
			fmt.Printf("  Installed: %s\n", model.InstalledAt.Format("2006-01-02 15:04:05"))
		}
		
		return nil
	},
}

var uninstallCmd = &cobra.Command{
	Use:   "uninstall [model-name]",
	Short: "Uninstall a model",
	Long:  `Remove an installed model and update the YAML configuration.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		modelName := args[0]
		
		fmt.Printf("Uninstalling model: %s\n", modelName)
		
		// Remove from filesystem
		modelInstaller := installer.NewModelInstaller(installModelsDir)
		if err := modelInstaller.Uninstall(modelName); err != nil {
			return fmt.Errorf("failed to uninstall model: %w", err)
		}
		
		fmt.Println("Model removed from filesystem")
		
		// Generate updated YAML configuration
		fmt.Println("\nUpdating YAML configuration...")
		yamlGen := config.NewYAMLGenerator(installModelsDir, installedModelsYAMLPath)
		if err := yamlGen.GenerateYAML(); err != nil {
			fmt.Printf("Warning: Failed to generate YAML: %v\n", err)
		} else {
			fmt.Printf("Updated configuration: %s\n", yamlGen.GetOutputPath())
		}
		
		fmt.Println("\nUninstallation complete!")
		return nil
	},
}

var generateYAMLCmd = &cobra.Command{
	Use:   "generate-yaml",
	Short: "Generate YAML configuration for installed models",
	Long: `Generate a YAML configuration file for all installed models.
The generated file matches the structure of edge-router/config.with-model-matching.yaml
and includes model capabilities and matching rules.

You can then manually copy the models you want into your edge-router config.yaml.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Generating YAML configuration for installed models...")
		
		yamlGen := config.NewYAMLGenerator(installModelsDir, installedModelsYAMLPath)
		if err := yamlGen.GenerateYAML(); err != nil {
			return fmt.Errorf("failed to generate YAML: %w", err)
		}
		
		fmt.Printf("\nGenerated configuration: %s\n", yamlGen.GetOutputPath())
		fmt.Println("\nYou can now copy the models you want into your edge-router config.yaml")
		fmt.Println("The structure matches edge-router/config.with-model-matching.yaml")
		
		return nil
	},
}

// createModelInstaller creates a model installer with appropriate converters
func createModelInstaller(modelsDir string, sampleModel *types.ModelMetadata) *installer.ModelInstaller {
	inst := installer.NewModelInstaller(modelsDir)
	
	// Add MLX converter on macOS
	if runtime.GOOS == "darwin" {
		mlxConv := converter.NewMLXConverter()
		if mlxConv.IsAvailable() == nil {
			inst.AddConverter(mlxConv)
		}
	}
	
	// Add GGUF converter (placeholder for now)
	// ggufConv := converter.NewGGUFConverter()
	// inst.AddConverter(ggufConv)
	
	return inst
}

// buildUserNeeds constructs UserNeeds from CLI flags with sensible defaults
func buildUserNeeds(tasks []string, quality, privacy string, storageGB, latencyMS int, responsivenessLevel string) types.UserNeeds {
	// Use default tasks if none specified
	if len(tasks) == 0 {
		tasks = []string{"general", "chat"}
	}
	
	// Convert responsiveness level to latency tolerance if provided
	// This provides a more user-friendly interface
	if responsivenessLevel != "" {
		switch responsivenessLevel {
		case "interactive", "realtime", "fast":
			latencyMS = 500 // Low latency - prioritize speed
		case "balanced", "normal", "medium":
			latencyMS = 2000 // Medium latency - balanced
		case "batch", "background", "slow":
			latencyMS = 10000 // High latency - prioritize quality over speed
		default:
			// Keep existing latencyMS value or use default
			if latencyMS == 0 {
				latencyMS = 1000 // Default to 1 second
			}
		}
	} else if latencyMS == 0 {
		// No responsiveness or latency specified, use default
		latencyMS = 1000
	}
	
	// Build task scores (equal weight for all specified tasks)
	taskScores := make(map[string]float64)
	weight := 1.0
	for i, task := range tasks {
		// First task gets full weight, others get decreasing weight
		if i == 0 {
			taskScores[task] = 1.0
		} else {
			weight *= 0.8
			taskScores[task] = weight
		}
	}
	
	// Build domain scores from tasks
	domainScores := make(map[string]float64)
	for _, task := range tasks {
		switch task {
		case "code", "coding", "programming":
			domainScores["technical"] = 1.0
			domainScores["code"] = 1.0
		case "reasoning", "analysis", "research":
			domainScores["technical"] = 0.9
			domainScores["reasoning"] = 1.0
		case "writing", "creative":
			domainScores["creative"] = 1.0
		case "chat", "conversation":
			domainScores["general"] = 1.0
		default:
			domainScores["general"] = 1.0
		}
	}
	
	// Ensure general domain has a score
	if _, ok := domainScores["general"]; !ok {
		domainScores["general"] = 0.8
	}
	
	return types.UserNeeds{
		Declared: types.DeclaredPreferences{
			PrimaryTasks:       tasks,
			QualityPreference:  quality,
			PrivacyRequirement: privacy,
			StorageLimitGB:     storageGB,
			LatencyToleranceMS: latencyMS,
		},
		Combined: types.CombinedScores{
			Tasks:   taskScores,
			Domains: domainScores,
		},
	}
}

func init() {
	// Add commands
	rootCmd.AddCommand(profileCmd)
	rootCmd.AddCommand(recommendCmd)
	rootCmd.AddCommand(recommendJSONCmd)
	rootCmd.AddCommand(selectCmd)
	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(uninstallCmd)
	rootCmd.AddCommand(generateYAMLCmd)
	
	// Add optional flags to recommend command
	recommendCmd.Flags().StringSliceVarP(&tasks, "tasks", "t", []string{},
		"Primary tasks (optional, comma-separated): general,chat,code,reasoning,writing,creative")
	recommendCmd.Flags().StringVarP(&qualityPref, "quality", "q", "medium",
		"Quality preference (optional): low,medium,high,premium")
	recommendCmd.Flags().StringVarP(&privacyReq, "privacy", "p", "local_preferred",
		"Privacy requirement (optional): local_only,local_preferred,cloud_acceptable")
	recommendCmd.Flags().IntVarP(&storageLimitGB, "storage", "s", 100,
		"Storage limit in GB (optional)")
	recommendCmd.Flags().StringVarP(&responsiveness, "responsiveness", "r", "",
		"Responsiveness level (optional): interactive,balanced,batch")
	recommendCmd.Flags().IntVarP(&latencyToleranceMS, "latency", "l", 0,
		"Latency tolerance in milliseconds (optional, advanced)")
	
	// Add same optional flags to recommend-json command
	recommendJSONCmd.Flags().StringSliceVarP(&tasks, "tasks", "t", []string{},
		"Primary tasks (optional, comma-separated): general,chat,code,reasoning,writing,creative")
	recommendJSONCmd.Flags().StringVarP(&qualityPref, "quality", "q", "medium",
		"Quality preference (optional): low,medium,high,premium")
	recommendJSONCmd.Flags().StringVarP(&privacyReq, "privacy", "p", "local_preferred",
		"Privacy requirement (optional): local_only,local_preferred,cloud_acceptable")
	recommendJSONCmd.Flags().IntVarP(&storageLimitGB, "storage", "s", 100,
		"Storage limit in GB (optional)")
	recommendJSONCmd.Flags().StringVarP(&responsiveness, "responsiveness", "r", "",
		"Responsiveness level (optional): interactive,balanced,batch")
	recommendJSONCmd.Flags().IntVarP(&latencyToleranceMS, "latency", "l", 0,
		"Latency tolerance in milliseconds (optional, advanced)")
	
	// Add flags to select command
	selectCmd.Flags().StringSliceVarP(&tasks, "tasks", "t", []string{},
		"Primary tasks (optional, comma-separated): general,chat,code,reasoning,writing,creative")
	selectCmd.Flags().StringVarP(&qualityPref, "quality", "q", "medium",
		"Quality preference (optional): low,medium,high,premium")
	selectCmd.Flags().StringVarP(&privacyReq, "privacy", "p", "local_preferred",
		"Privacy requirement (optional): local_only,local_preferred,cloud_acceptable")
	selectCmd.Flags().StringVarP(&responsiveness, "responsiveness", "r", "",
		"Responsiveness level (optional): interactive,balanced,batch")
	selectCmd.Flags().StringVarP(&installFormat, "format", "f", "",
		"Model format (optional): mlx,gguf,safetensors")
	selectCmd.Flags().StringVar(&installQuantization, "quantization", "",
		"Quantization level (optional): q4,q8,fp16")
	selectCmd.Flags().StringVar(&installModelsDir, "models-dir", "",
		"Directory to install models (optional, default: ~/.llm-d-edge/models)")
	selectCmd.Flags().IntVar(&installPriority, "priority", 1,
		"Starting priority for installed models")
	selectCmd.Flags().StringVar(&installedModelsYAMLPath, "yaml-output", "",
		"Path for generated YAML file (optional, default: models-dir/../installed-models.yaml)")
	
	// Add flags to install command
	installCmd.Flags().StringVarP(&installFormat, "format", "f", "mlx",
		"Model format: mlx,gguf,safetensors")
	installCmd.Flags().StringVar(&installQuantization, "quantization", "q4",
		"Quantization level: q4,q8,fp16")
	installCmd.Flags().StringVar(&installModelsDir, "models-dir", "",
		"Directory to install models (optional, default: ~/.llm-d-edge/models)")
	installCmd.Flags().IntVar(&installPriority, "priority", 1,
		"Priority for this model in generated YAML")
	installCmd.Flags().StringVar(&installedModelsYAMLPath, "yaml-output", "",
		"Path for generated YAML file (optional, default: models-dir/../installed-models.yaml)")
	
	// Add flags to list command
	listCmd.Flags().StringVar(&installModelsDir, "models-dir", "",
		"Directory where models are installed (optional, default: ~/.llm-d-edge/models)")
	
	// Add flags to uninstall command
	uninstallCmd.Flags().StringVar(&installModelsDir, "models-dir", "",
		"Directory where models are installed (optional, default: ~/.llm-d-edge/models)")
	uninstallCmd.Flags().StringVar(&installedModelsYAMLPath, "yaml-output", "",
		"Path for generated YAML file (optional, default: models-dir/../installed-models.yaml)")
	
	// Add flags to generate-yaml command
	generateYAMLCmd.Flags().StringVar(&installModelsDir, "models-dir", "",
		"Directory where models are installed (optional, default: ~/.llm-d-edge/models)")
	generateYAMLCmd.Flags().StringVar(&installedModelsYAMLPath, "yaml-output", "",
		"Path for generated YAML file (optional, default: models-dir/../installed-models.yaml)")
}

// Made with Bob
