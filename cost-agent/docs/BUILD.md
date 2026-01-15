# Quick Guide: Build and Push for AMD64

## Quick Start

```bash
# 1. Set your GitHub token (get it from GitHub Settings → Developer settings → Personal access tokens)
export GHCR_TOKEN="ghp_your_token_here"

# 2. Build and push for AMD64
make release PLATFORM=linux/amd64
```

That's it! The image will be built for AMD64 and pushed to `ghcr.io/bugfreev587/cost-agent:v1.0.8`

## To Use a Different Version

```bash
make release PLATFORM=linux/amd64 IMAGE_TAG=v1.0.9
```

## Step-by-Step (If you prefer separate commands)

```bash
# 1. Set GitHub token
export GHCR_TOKEN="ghp_your_token_here"

# 2. Build for AMD64
make build PLATFORM=linux/amd64

# 3. Push to GHCR
make push
```

## Get GitHub Token

1. Go to: https://github.com/settings/tokens
2. Click "Generate new token (classic)"
3. Select scopes: `write:packages`, `read:packages`
4. Generate and copy the token

## Verify

After pushing, check: https://github.com/bugfreev587/cost-agent/pkgs/container/cost-agent

For detailed instructions, see: [docs/build-and-push.md](docs/build-and-push.md)

