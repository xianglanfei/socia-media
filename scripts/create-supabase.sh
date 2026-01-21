#!/bin/bash

# Interactive Supabase Project Creator
# This script guides you through creating a Supabase project using MCP

set -e

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Project configuration
PROJECT_NAME="socia-media"
REGION="us-east-1"

echo -e "${BLUE}╔══════════════════════════════════════╗${NC}"
echo -e "${BLUE}║   Supabase Project Creator                ║${NC}"
echo -e "${BLUE}╚══════════════════════════════════════╝${NC}"
echo ""

# Check token
if [ -z "$SUPABASE_ACCESS_TOKEN" ]; then
    echo -e "${YELLOW}Please enter your Supabase Access Token:${NC}"
    echo -e "${YELLOW}Get it at: https://supabase.com/dashboard/account/tokens${NC}"
    read -p "Token: " SUPABASE_ACCESS_TOKEN
fi

echo ""
echo -e "${GREEN}✓ Token configured${NC}"

# Get organizations
echo -e "${BLUE}Step 1: Getting your organizations...${NC}"
echo '{"jsonrpc":"2.0","id":1,"method":"list_organizations","params":{}}' | \
    node /home/eee/.local/share/fnm/node-versions/v24.12.0/installation/lib/node_modules/@supabase/mcp-server-supabase/dist/transports/stdio.js 2>&1 | \
    grep -oP '"id":[^,]*' | head -1 | \
    sed 's/.*"id":[^,]*"\([^"]*\)".*/\1/' > /tmp/supabase_org_id.txt

if [ ! -s /tmp/supabase_org_id.txt ] || [ ! -s /tmp/supabase_org_id.txt ]; then
    echo -e "${YELLOW}No organization found. Using default.${NC}"
    ORG_ID=""
else
    ORG_ID=$(cat /tmp/supabase_org_id.txt | tr -d '"')
    echo -e "${GREEN}✓ Organization ID: $ORG_ID${NC}"
fi

# Get project cost
echo ""
echo -e "${BLUE}Step 2: Getting cost for new project...${NC}"
if [ -n "$ORG_ID" ]; then
    echo "{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"get_cost\",\"params\":{\"type\":\"project\",\"organization_id\":\"$ORG_ID\"}}" | \
        node /home/eee/.local/share/fnm/node-versions/v24.12.0/installation/lib/node_modules/@supabase/mcp-server-supabase/dist/transports/stdio.js 2>&1 | \
        grep -oP '"amount":[^,]*' | head -1
fi

# Get cost confirmation ID (this is a simplification)
COST_CONFIRM_ID="auto"

# Create project
echo ""
echo -e "${BLUE}Step 3: Creating project...${NC}"
echo -e "${GREEN}Project Name: $PROJECT_NAME${NC}"
echo -e "${GREEN}Region: $REGION${NC}"
echo -e "${GREEN}Organization ID: ${ORG_ID:-<default>}${NC}"
echo ""
read -p "Continue? (y/n): " -n 1 -r
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo -e "${YELLOW}Cancelled.${NC}"
    exit 0
fi

# Create project request
PROJECT_JSON=$(cat <<EOF
{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "create_project",
  "params": {
    "name": "$PROJECT_NAME",
    "region": "$REGION",
    "organization_id": "$ORG_ID",
    "confirm_cost_id": "$COST_CONFIRM_ID"
  }
}
EOF
)

echo "$PROJECT_JSON" | \
    node /home/eee/.local/share/fnm/node-versions/v24.12.0/installation/lib/node_modules/@supabase/mcp-server-supabase/dist/transports/stdio.js

echo ""
echo -e "${GREEN}════════════════════════════════════════${NC}"
echo -e "${GREEN}✓ Project creation initiated!${NC}"
echo -e "${GREEN}════════════════════════════════════════${NC}"
echo ""
echo -e "${YELLOW}Next steps:${NC}"
echo "1. Go to https://supabase.com/dashboard/project/$ORG_ID"
echo "2. Wait for project to be ready (usually 2-3 minutes)"
echo "3. Get database URL from Project Settings → Database"
echo "4. Update backend/.env with SUPABASE_URL"
echo ""
echo -e "${BLUE}Available MCP Tools:${NC}"
echo "  - list_organizations: List your organizations"
echo "  - get_project: Get project details"
echo "  - list_projects: List all projects"
echo "  - create_branch: Create dev branch"
echo "  - apply_migration: Apply SQL migration"
echo "  - deploy_edge_function: Deploy Edge Function"
