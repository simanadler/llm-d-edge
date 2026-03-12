Recommended Model Setup for Your M4 Max MacBook Pro (64GB RAM)
With your powerful M4 Max and 64GB RAM, you can run larger, high-quality models. Here are my recommendations:

Best Models for Your System:
Qwen2.5-7B-Instruct (Recommended - Best balance)

Size: ~4-5GB (4-bit quantized)
Performance: Excellent quality, fast inference
Use case: General purpose, coding, reasoning
Qwen2.5-14B-Instruct (High quality)

Size: ~8-10GB (4-bit quantized)
Performance: Superior quality, still fast on M4 Max
Use case: Complex tasks, better reasoning
Llama-3.1-8B-Instruct (Alternative)

Size: ~5GB (4-bit quantized)
Performance: Very good, Meta's latest
Use case: General purpose
Setup Instructions:
Step 1: Install MLX

pip3 install mlx-lm

Step 2: Download and Convert Model (Qwen2.5-7B recommended)

# Create models directory
mkdir -p ~/Library/Application\ Support/llm-d/models

# Download and convert to MLX format with 4-bit quantization
python3 -m mlx_lm.convert \
  --hf-path Qwen/Qwen2.5-7B-Instruct \
  --mlx-path ~/Library/Application\ Support/llm-d/models/Qwen--Qwen2.5-7B-Instruct \
  --quantize \
  --q-bits 4

Step 3: Update your config.yaml

edge:
  platform: macos
  routing:
    policy: local-first
    fallback: remote
  models:
    local:
      - name: "Qwen/Qwen2.5-7B-Instruct"
        path: "~/Library/Application Support/llm-d/models/Qwen--Qwen2.5-7B-Instruct"
        format: "mlx"
        quantization: "4bit"

Step 4: Test the model

# Test with MLX directly
python3 -m mlx_lm.generate \
  --model ~/Library/Application\ Support/llm-d/models/Qwen--Qwen2.5-7B-Instruct \
  --prompt "Hello, how are you?" \
  --max-tokens 100

Expected Performance on M4 Max:
Qwen2.5-7B: 60-80 tokens/sec, ~200ms first token
Qwen2.5-14B: 40-50 tokens/sec, ~300ms first token
Memory usage: 4-10GB depending on model size
Why These Models?
Qwen2.5 series: State-of-the-art open models, excellent at coding and reasoning
MLX optimized: Takes full advantage of your M4 Max's unified memory and Neural Engine
4-bit quantization: Minimal quality loss, 4x smaller, much faster
Your M4 Max can easily handle even the 14B model with excellent performance!