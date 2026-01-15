# Building and Pushing cost-agent for AMD64 to GitHub Packages

This guide explains how to build the cost-agent Docker image for AMD64 architecture and push it to GitHub Container Registry (GHCR).

## Prerequisites

1. **Docker** installed and running
2. **Docker Buildx** enabled (usually comes with Docker Desktop)
3. **GitHub Personal Access Token (PAT)** with package write permissions
4. **GitHub username**: `bugfreev587` (or update in Makefile)

## Step 1: Get GitHub Personal Access Token

1. Go to GitHub → Settings → Developer settings → Personal access tokens → Tokens (classic)
2. Click "Generate new token (classic)"
3. Give it a name (e.g., "GHCR Push Token")
4. Select scopes:
   - ✅ `write:packages` - Upload packages to GitHub Package Registry
   - ✅ `read:packages` - Download packages from GitHub Package Registry
   - ✅ `delete:packages` - Delete packages (optional, for cleanup)
5. Click "Generate token"
6. **Copy the token immediately** (you won't see it again)

## Step 2: Set Environment Variable

Set your GitHub token as an environment variable:

```bash
export GHCR_TOKEN="your_github_personal_access_token_here"
```

Or add it to your `~/.bashrc` or `~/.zshrc`:

```bash
echo 'export GHCR_TOKEN="your_token_here"' >> ~/.zshrc
source ~/.zshrc
```

## Step 3: Build for AMD64

### Option A: Using Makefile (Recommended)

The Makefile supports building for specific platforms. To build for AMD64:

```bash
# Build for AMD64 explicitly
make build PLATFORM=linux/amd64
```

Or set it as an environment variable:

```bash
export PLATFORM=linux/amd64
make build
```

### Option B: Using Docker Buildx Directly

```bash
# Build for AMD64
docker buildx build --platform linux/amd64 \
  -t ghcr.io/bugfreev587/cost-agent:v1.0.8 \
  -f Dockerfile \
  .
```

### Option C: Using Docker Build (Simple, but slower on non-AMD64 machines)

```bash
docker build --platform linux/amd64 \
  -t ghcr.io/bugfreev587/cost-agent:v1.0.8 \
  -f Dockerfile \
  .
```

## Step 4: Push to GitHub Container Registry

### Option A: Using Makefile (Recommended)

```bash
# Set the tag if you want to use a different version
export IMAGE_TAG=v1.0.8

# Login and push (Makefile handles both)
make push
```

Or combine build and push:

```bash
# Build and push in one command
make release PLATFORM=linux/amd64
```

### Option B: Manual Steps

1. **Login to GHCR:**

   ```bash
   echo $GHCR_TOKEN | docker login ghcr.io -u bugfreev587 --password-stdin
   ```

2. **Push the image:**

   ```bash
   docker push ghcr.io/bugfreev587/cost-agent:v1.0.8
   ```

## Step 5: Verify the Push

1. Go to: `https://github.com/bugfreev587/cost-agent/pkgs/container/cost-agent`
2. You should see your image tag listed
3. Check the architecture in the package details (should show `linux/amd64`)

## Complete Example Workflow

Here's a complete example for building and pushing version `v1.0.9`:

```bash
# Navigate to project directory
cd /path/to/cost-agent

# Set your GitHub token (one-time setup)
export GHCR_TOKEN="ghp_your_token_here"

# Build and push for AMD64
make release PLATFORM=linux/amd64 IMAGE_TAG=v1.0.9
```

## Updating the Image Tag

To push a new version, update the `IMAGE_TAG` variable:

```bash
# Method 1: Environment variable
export IMAGE_TAG=v1.0.9
make release PLATFORM=linux/amd64

# Method 2: Inline
make release PLATFORM=linux/amd64 IMAGE_TAG=v1.0.9

# Method 3: Edit Makefile directly
# Change line 6: IMAGE_TAG ?= v1.0.9
make release PLATFORM=linux/amd64
```

## Troubleshooting

### Error: "denied: permission_denied"

**Solution:** Your GitHub token doesn't have the correct permissions. Make sure it has `write:packages` scope.

### Error: "unauthorized: authentication required"

**Solution:** You're not logged in. Run:
```bash
echo $GHCR_TOKEN | docker login ghcr.io -u bugfreev587 --password-stdin
```

### Error: "no matching manifest for linux/amd64"

**Solution:** Make sure you're using `docker buildx` or specify `--platform linux/amd64` explicitly.

### Building on Apple Silicon (M1/M2/M3)

If you're on Apple Silicon and want to build for AMD64:

```bash
# Create a buildx builder (one-time setup)
docker buildx create --name multiplatform --use

# Build for AMD64
docker buildx build --platform linux/amd64 \
  -t ghcr.io/bugfreev587/cost-agent:v1.0.8 \
  --load \
  -f Dockerfile \
  .
```

Or use the Makefile which handles this automatically when you set `PLATFORM=linux/amd64`.

### Using Docker Buildx for Multi-arch (Advanced)

To build and push for multiple architectures:

```bash
# Build and push for both AMD64 and ARM64
docker buildx build --platform linux/amd64,linux/arm64 \
  -t ghcr.io/bugfreev587/cost-agent:v1.0.8 \
  --push \
  -f Dockerfile \
  .
```

## Making the Package Public (Optional)

By default, packages are private. To make them public:

1. Go to: `https://github.com/bugfreev587/cost-agent/pkgs/container/cost-agent`
2. Click "Package settings"
3. Scroll down to "Danger Zone"
4. Click "Change visibility" → "Make public"

Or use GitHub CLI:

```bash
gh api \
  -X PATCH \
  /orgs/bugfreev587/packages/container/cost-agent \
  -f visibility=public
```

## Using the Image

After pushing, you can pull and use the image:

```bash
# Pull the image
docker pull ghcr.io/bugfreev587/cost-agent:v1.0.8

# Run the container
docker run --rm \
  -e AGENT_SERVER_URL=http://localhost:8080 \
  -e AGENT_API_KEY=your_key_id:your_secret \
  ghcr.io/bugfreev587/cost-agent:v1.0.8
```

In Kubernetes, reference it as:

```yaml
image: ghcr.io/bugfreev587/cost-agent:v1.0.8
```

If the package is private, you'll need to create a Kubernetes secret with your GitHub token for image pulling.

