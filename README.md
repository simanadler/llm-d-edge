Welcome to the LLM-D Edge Project (final name pending)

The aim of this project is to enable inferencing to occur either locally on edge devices (personal computers, laptops, smartphones, tablets, etc.) or remotely on the cloud without the user having to mnaually manage the local models nor the routing of the inferencing requests.

[Proposal](./docs/llm-d-edge-proposal.md) - including benefits and review of existing solutions

[Proposed Architecture](./docs/cross-platform-llm-d-edge-architectur.md)

An initial version of the router has been implemented in the [edge-router](./edge-router) directory (MacOS local implementation only).  It supports:
* Routing of inference requests to local or remote models as described in the [README's routing policies](./edge-router/README.md#routing-policies)
* Ability to route to an alternative local model if the exact model requested is not available locally, as described in [the model matching description](./edge-router/MODEL_MATCHING.md)


