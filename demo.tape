# Klotski Puzzle — Game Mechanics Demo
# Run: vhs demo.tape
# Requires: go (to build), vhs (https://github.com/charmbracelet/vhs)

Output demo.gif

Require ./puzzle

Set Shell "bash"
Set FontSize 22
Set Width 1200
Set Height 600
Set Padding 20
Set TypingSpeed 50ms
Set Theme "Catppuccin Mocha"
Set WindowBar Colorful

# Launch the game (binary must be pre-built with: go build -o puzzle .)
Type "./puzzle"
Enter
Sleep 3s

# --- Navigate the board ---
# Cursor starts at (0,0). Move around to show the board.
Down
Sleep 300ms
Down
Sleep 300ms
Right
Sleep 300ms
Right
Sleep 300ms
Down
Sleep 300ms

# --- Select and move a piece ---
# Select whatever piece is under the cursor
Enter
Sleep 600ms

# Try moving the piece in different directions
Left
Sleep 500ms
Up
Sleep 500ms

# Deselect
Escape
Sleep 600ms

# --- Move another piece ---
Up
Sleep 300ms
Up
Sleep 300ms
Left
Sleep 300ms

# Select and move
Enter
Sleep 500ms
Down
Sleep 500ms
Right
Sleep 500ms
Escape
Sleep 600ms

# --- Undo moves ---
Type "u"
Sleep 400ms
Type "u"
Sleep 400ms
Type "u"
Sleep 400ms
Sleep 500ms

# --- Toggle cheat mode ---
# Shows optimal next move with purple highlight and direction arrow
Type "?"
Sleep 2500ms

# Turn off cheat mode
Type "?"
Sleep 600ms

# --- Switch difficulty levels ---
# Easy
Type "1"
Sleep 1500ms

# Medium
Type "2"
Sleep 1500ms

# Hard
Type "3"
Sleep 1500ms

# --- Toggle coordinate labels ---
Type "c"
Sleep 1500ms
Type "c"
Sleep 500ms

# Quit
Type "q"
Sleep 500ms
