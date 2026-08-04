#!/bin/bash

# This script creates the distribution ZIP file for end users
# Run this ONCE to create the distributable package

set -e

echo "🎯 Creating Indieclaw Distribution Package..."
echo ""

# Get the script directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# Create distribution directory
DIST_DIR="$SCRIPT_DIR/dist/indieclaw-installer"
mkdir -p "$DIST_DIR"

echo "📋 Copying installer files..."

# Copy main files
cp "$SCRIPT_DIR/install.sh" "$DIST_DIR/"
cp "$SCRIPT_DIR/README_DISTRIBUTION.md" "$DIST_DIR/README.md"
cp "$SCRIPT_DIR/00_START_HERE.md" "$DIST_DIR/"
cp "$SCRIPT_DIR/QUICK_START.txt" "$DIST_DIR/"
cp "$SCRIPT_DIR/INSTALLATION.md" "$DIST_DIR/"
cp "$SCRIPT_DIR/OLLAMA_MODELS_GUIDE.md" "$DIST_DIR/"
cp "$SCRIPT_DIR/INSTALLER_README.md" "$DIST_DIR/"

# Create simple setup instructions
cat > "$DIST_DIR/SETUP_INSTRUCTIONS.txt" << 'EOF'
================================================================================
                    INDIECLAW INSTALLATION - START HERE
================================================================================

WHAT IS INDIECLAW?

A personal AI assistant for your Mac that:
  ✅ Runs completely on your laptop (private)
  ✅ Integrates with WhatsApp
  ✅ Has a customizable personality
  ✅ Requires no cloud subscriptions
  ✅ Needs no coding knowledge

================================================================================
                       SYSTEM REQUIREMENTS CHECK
================================================================================

Before you start, verify:

  1. Do you have a Mac with Apple Silicon? (M1, M2, M3, M4)
     → Open Apple menu → About This Mac
     → Look for "Apple Silicon" or "M1/M2/M3/M4"

  2. Do you have at least 16GB of RAM?
     → Same location: About This Mac
     → Look for "Memory"

  3. Do you have at least 80GB of free disk space?
     → Open Finder → About This Mac → Storage
     → Check available space

  4. Do you have an internet connection?

If ALL 4 are YES, continue below.

If any are NO, you cannot install Indieclaw.

================================================================================
                        3-STEP INSTALLATION
================================================================================

STEP 1: Open Terminal App
────────────────────────────────────────────────────────────────────────────

1. Press Cmd + Space (or Fn + Space)
2. Type: Terminal
3. Press Enter
4. A black window opens


STEP 2: Navigate to This Folder
────────────────────────────────────────────────────────────────────────────

In Terminal, type:

    cd ~/Downloads/indieclaw-installer

(Or drag this folder into Terminal after typing "cd ")


STEP 3: Run the Installer
────────────────────────────────────────────────────────────────────────────

In Terminal, type exactly:

    bash install.sh

Press Enter.

The installer will:
  • Ask a few questions
  • Download tools and AI models (~20GB)
  • Configure everything automatically
  • Show you what to do next

⏱️  This takes about 60 minutes. Don't interrupt it.


================================================================================
                        WHAT HAPPENS NEXT
================================================================================

After the installer finishes, you'll see colorful instructions.

They tell you to:

1. Open 3-4 Terminal windows
2. Copy/paste simple commands
3. Scan a QR code with WhatsApp
4. Start chatting with your AI!

Just follow the colored instructions on screen.


================================================================================
                     TROUBLESHOOTING
================================================================================

"Command not found: bash"
→ You're in the wrong directory
→ Try: cd ~/Downloads/indieclaw-installer

"Permission denied"
→ Type this first: chmod +x install.sh

"Installation hangs"
→ It's downloading AI models (~20GB)
→ Don't interrupt. Wait 15-20 minutes.

"Port already in use"
→ Close other applications
→ Try again


For more help, see:
  • README.md (orientation)
  • INSTALLATION.md (detailed guide)
  • QUICK_START.txt (daily commands)


================================================================================
                        YOU'RE READY!
================================================================================

Ready to begin? In Terminal, type:

    bash install.sh

That's it! The installer handles everything else.

Questions? See the documentation files in this folder.

Enjoy your personal AI assistant! 🚀

================================================================================
EOF

# Make installer executable
chmod +x "$DIST_DIR/install.sh"
chmod +x "$DIST_DIR/SETUP_INSTRUCTIONS.txt"

echo "📦 Creating ZIP file..."

# Create ZIP
cd "$SCRIPT_DIR/dist"
rm -f indieclaw-installer.zip
zip -r -q indieclaw-installer.zip indieclaw-installer/

# Show results
ZIP_SIZE=$(ls -lh indieclaw-installer.zip | awk '{print $5}')
FILE_COUNT=$(ls -1 "$DIST_DIR" | wc -l)

echo ""
echo "✅ Distribution package created!"
echo ""
echo "📊 Package Details:"
echo "   Location: $SCRIPT_DIR/dist/indieclaw-installer.zip"
echo "   Size: $ZIP_SIZE"
echo "   Files: $FILE_COUNT"
echo ""
echo "📋 Contents:"
ls -1 "$DIST_DIR" | sed 's/^/   • /'
echo ""
echo "════════════════════════════════════════════════════════════"
echo ""
echo "🎯 Ready to distribute!"
echo ""
echo "Share this file with end users:"
echo "   indieclaw-installer.zip (~$ZIP_SIZE)"
echo ""
echo "They just need to:"
echo "   1. Unzip the file"
echo "   2. Read SETUP_INSTRUCTIONS.txt"
echo "   3. Run: bash install.sh"
echo ""
echo "No git clone needed!"
echo "No code repositories needed!"
echo "No technical knowledge needed!"
echo ""
echo "════════════════════════════════════════════════════════════"
