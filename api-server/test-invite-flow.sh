#!/bin/bash

# Test script for invite flow
# Usage: ./test-invite-flow.sh <your-user-id> <email-to-invite>

API_URL="${API_URL:-https://api-server-production-7a9d.up.railway.app}"
USER_ID="$1"
INVITE_EMAIL="$2"

if [ -z "$USER_ID" ] || [ -z "$INVITE_EMAIL" ]; then
    echo "Usage: ./test-invite-flow.sh <your-user-id> <email-to-invite>"
    echo ""
    echo "Example: ./test-invite-flow.sh user_2abc123 test@example.com"
    echo ""
    echo "You can find your user ID in the Profile page or browser console."
    exit 1
fi

echo "=== Testing Invite Flow ==="
echo "API URL: $API_URL"
echo "Your User ID: $USER_ID"
echo "Email to invite: $INVITE_EMAIL"
echo ""

# Step 1: List current users
echo "--- Step 1: List current users in your tenant ---"
curl -s -X GET "$API_URL/v1/users" \
    -H "Content-Type: application/json" \
    -H "X-User-Id: $USER_ID" | jq .
echo ""

# Step 2: Invite a new user
echo "--- Step 2: Invite user: $INVITE_EMAIL ---"
INVITE_RESPONSE=$(curl -s -X POST "$API_URL/v1/admin/users/invite" \
    -H "Content-Type: application/json" \
    -H "X-User-Id: $USER_ID" \
    -d "{\"email\": \"$INVITE_EMAIL\", \"name\": \"Test User\", \"role\": \"viewer\"}")
echo "$INVITE_RESPONSE" | jq .
echo ""

# Step 3: List users again to see the pending invitation
echo "--- Step 3: List users again (should see pending invitation) ---"
curl -s -X GET "$API_URL/v1/users" \
    -H "Content-Type: application/json" \
    -H "X-User-Id: $USER_ID" | jq .
echo ""

# Step 4: Simulate the invited user signing up
echo "--- Step 4: Simulate invited user signup via auth/sync ---"
# Use a fake Clerk ID for testing
FAKE_CLERK_ID="user_test_$(date +%s)"
SYNC_RESPONSE=$(curl -s -X POST "$API_URL/v1/auth/sync" \
    -H "Content-Type: application/json" \
    -d "{\"clerk_user_id\": \"$FAKE_CLERK_ID\", \"email\": \"$INVITE_EMAIL\", \"first_name\": \"Test\", \"last_name\": \"User\"}")
echo "$SYNC_RESPONSE" | jq .
echo ""

# Check if the user was added to the same tenant
SYNC_TENANT_ID=$(echo "$SYNC_RESPONSE" | jq -r '.tenant_id')
SYNC_ROLE=$(echo "$SYNC_RESPONSE" | jq -r '.role')

echo "--- Result Analysis ---"
echo "Invited user tenant_id: $SYNC_TENANT_ID"
echo "Invited user role: $SYNC_ROLE"

if [ "$SYNC_ROLE" = "viewer" ]; then
    echo "SUCCESS: Invited user has the correct role (viewer)"
else
    echo "WARNING: Expected role 'viewer' but got '$SYNC_ROLE'"
fi

# Step 5: List users one more time to confirm
echo ""
echo "--- Step 5: Final user list (invited user should be active) ---"
curl -s -X GET "$API_URL/v1/users" \
    -H "Content-Type: application/json" \
    -H "X-User-Id: $USER_ID" | jq .
echo ""

echo "=== Test Complete ==="
