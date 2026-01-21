#!/bin/bash

# This script uses Supabase MCP to create a project programmatically
# You need to set your SUPABASE_ACCESS_TOKEN first

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check for SUPABASE_ACCESS_TOKEN
if [ -z "$SUPABASE_ACCESS_TOKEN" ]; then
    echo -e "${RED}Error: SUPABASE_ACCESS_TOKEN environment variable is not set${NC}"
    echo -e "${YELLOW}Get your token at: https://supabase.com/dashboard/account/tokens${NC}"
    exit 1
fi

# Configuration
PROJECT_NAME="socia-media"
REGION="us-east-1"  # Change this to your preferred region

echo -e "${GREEN}======================================${NC}"
echo -e "${GREEN}Supabase Project Creator${NC}"
echo -e "${GREEN}======================================${NC}"
echo ""

# Function to send MCP request
send_mcp_request() {
    local method="$1"
    local params="$2"

    echo -e "${YELLOW}Calling: $method${NC}"

    RESPONSE=$(echo "$params" | node /home/eee/.local/share/fnm/node-versions/v24.12.0/installation/lib/node_modules/@supabase/mcp-server-supabase/dist/transports/stdio.js 2>&1)

    echo "$RESPONSE"
    echo ""
}

# Step 1: List organizations to get organization ID
echo -e "${GREEN}Step 1: Listing organizations...${NC}"
send_mcp_request "list_organizations" '{"jsonrpc":"2.0","id":1,"method":"list_organizations","params":{}}'

# You'll need to extract the organization ID from the response
# For now, let's assume we want to create in the first organization

# Step 2: Get cost of creating a project
echo -e "${GREEN}Step 2: Getting cost for new project...${NC}"
send_mcp_request "get_cost" "{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"get_cost\",\"params\":{\"type\":\"project\",\"organization_id\":\"YOUR_ORG_ID\"}}"

# Step 3: Confirm cost (required before create_project)
echo -e "${GREEN}Step 3: Confirming cost...${NC}"
send_mcp_request "confirm_cost" "{\"jsonrpc\":\"2.0\",\"id\":3,\"method\":\"confirm_cost\",\"params\":{\"type\":\"project\",\"organization_id\":\"YOUR_ORG_ID\",\"recurrence\":\"hourly\"}}"

# Step 4: Create the project
echo -e "${GREEN}Step 4: Creating project: $PROJECT_NAME${NC}"
PROJECT_PARAMS=$(cat <<EOF
{
  "jsonrpc": "2.0",
  "id": 4,
  "method": "create_project",
  "params": {
    "name": "$PROJECT_NAME",
    "region": "$REGION",
    "organization_id": "YOUR_ORG_ID",
    "confirm_cost_id": "COST_CONFIRMATION_ID_FROM_STEP_3"
  }
}
EOF
)

send_mcp_request "create_project" "$PROJECT_PARAMS"

echo -e "${GREEN}======================================${NC}"
echo -e "${GREEN}Project creation initiated!${NC}"
echo -e "${YELLOW}Note: Project creation can take a few minutes${NC}"
echo -e "${YELLOW}Use the 'get_project' tool to check status${NC}"
echo -e "${GREEN}======================================${NC}"

# Instructions
echo ""
echo -e "${YELLOW}Manual Steps:${NC}"
echo "1. Update 'YOUR_ORG_ID' in this script with your actual organization ID"
echo "2. Update 'COST_CONFIRMATION_ID' with the ID returned from confirm_cost"
echo "3. Run: bash scripts/create-supabase-project.sh"
echo ""
echo -e "${GREEN}Or use MCP directly with Claude Code:${NC}"
echo "1. Set: export SUPABASE_ACCESS_TOKEN=your_token"
echo "2. Ask Claude to call: list_organizations, get_cost, confirm_cost, create_project"
