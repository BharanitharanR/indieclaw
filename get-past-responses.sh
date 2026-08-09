#!/bin/bash

echo "📊 Analyzing past responses from logs..."
echo ""

# Get the two test messages we're interested in
MESSAGES=("test_1786024051889_25f88a9a" "test_1786024363728_5a37f082")

for MSG_ID in "${MESSAGES[@]}"; do
  echo "═══════════════════════════════════════════════════════"
  echo "📨 Message: $MSG_ID"
  echo "═══════════════════════════════════════════════════════"
  
  # Get the original request
  grep "Processing request \[$MSG_ID\]" logs/orchestrator.log | head -1
  
  # Get planner decision
  echo ""
  echo "🤔 Planner Analysis:"
  grep -A 1 "\[$MSG_ID\]" logs/orchestrator.log | grep "can_help\|confidence"
  
  # Get steps that were planned
  echo ""
  echo "🔧 Execution Steps:"
  grep -A 10 "Refining.*planner steps" logs/orchestrator.log | grep "\[$MSG_ID\]\|Step [0-9]:" | head -10
  
  # Get final result
  echo ""
  echo "📊 Results:"
  grep "Pipeline completed \[$MSG_ID\]" logs/orchestrator.log
  
  # Get response publish
  grep "Published response \[$MSG_ID\]" logs/orchestrator.log
  
  echo ""
  echo ""
done
