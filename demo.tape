# Klotski Puzzle — Game Mechanics Demo
# Run: vhs demo.tape
# Requires: go (to build), vhs (https://github.com/charmbracelet/vhs)

Output demo.gif

Set Shell "bash"
Set FontSize 16
Set Width 80
Set Height 28
Set Padding 10
Set TypingSpeed 50ms
Set Theme "Catppuccin Mocha"

# Build the binary (hidden from recording)
Hide
Type "go build -o puzzle . && clear"
Enter
Sleep 10s
Show

# Launch the game
Type "./puzzle"
Enter
Sleep 2s

# --- Navigate the board ---
# Cursor starts at (0,0). Move around to show the board.
Down
Sleep 250ms
Down
Sleep 250ms
Right
Sleep 250ms
Right
Sleep 250ms
Down
Sleep 250ms

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
Sleep 250ms
Up
Sleep 250ms
Left
Sleep 250ms

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
