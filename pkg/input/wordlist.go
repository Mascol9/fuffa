package input

import (
	"bufio"
	"os"
	"regexp"
	"strings"

	"github.com/Mascol9/fuffa/pkg/ffuf"
)

type WordlistInput struct {
	active   bool
	config   *ffuf.Config
	data     [][]byte
	position int
	keyword  string
}

func NewWordlistInput(keyword string, value string, conf *ffuf.Config) (*WordlistInput, error) {
	var wl WordlistInput
	wl.active = true
	wl.keyword = keyword
	wl.config = conf
	wl.position = 0
	var valid bool
	var err error
	// stdin?
	if value == "-" {
		// yes
		valid = true
	} else {
		// no
		valid, err = wl.validFile(value)
	}
	if err != nil {
		return &wl, err
	}
	if valid {
		err = wl.readFile(value)
	}
	return &wl, err
}

// Position will return the current position in the input list
func (w *WordlistInput) Position() int {
	return w.position
}

// SetPosition sets the current position of the inputprovider
func (w *WordlistInput) SetPosition(pos int) {
	w.position = pos
}

// ResetPosition resets the position back to beginning of the wordlist.
func (w *WordlistInput) ResetPosition() {
	w.position = 0
}

// Keyword returns the keyword assigned to this InternalInputProvider
func (w *WordlistInput) Keyword() string {
	return w.keyword
}

// Next will return a boolean telling if there's words left in the list
func (w *WordlistInput) Next() bool {
	return w.position < len(w.data)
}

// IncrementPosition will increment the current position in the inputprovider data slice
func (w *WordlistInput) IncrementPosition() {
	w.position += 1
}

// Value returns the value from wordlist at current cursor position
func (w *WordlistInput) Value() []byte {
	return w.data[w.position]
}

// Total returns the size of wordlist
func (w *WordlistInput) Total() int {
	return len(w.data)
}

// Active returns boolean if the inputprovider is active
func (w *WordlistInput) Active() bool {
	return w.active
}

// Enable sets the inputprovider as active
func (w *WordlistInput) Enable() {
	w.active = true
}

// Disable disables the inputprovider
func (w *WordlistInput) Disable() {
	w.active = false
}

// validFile checks that the wordlist file exists and can be read
func (w *WordlistInput) validFile(path string) (bool, error) {
	_, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	f.Close()
	return true, nil
}

// readFile reads the file line by line to a byte slice
func (w *WordlistInput) readFile(path string) error {
	var file *os.File
	var err error
	if path == "-" {
		file = os.Stdin
	} else {
		file, err = os.Open(path)
		if err != nil {
			return err
		}
	}
	defer file.Close()

	var data [][]byte
	var ok bool
	// Global map to track unique payloads across all lines and extensions
	seen := make(map[string]bool)
	reader := bufio.NewScanner(file)
	re := regexp.MustCompile(`(?i)%ext%`)
	linesRead := 0
	for reader.Scan() {
		// Check wordlist limit (0 means unlimited)
		if w.config.WordlistLimit > 0 && linesRead >= w.config.WordlistLimit {
			break
		}
		if w.config.DirSearchCompat && len(w.config.Extensions) > 0 {
			text := []byte(reader.Text())
			if re.Match(text) {
				for _, ext := range w.config.Extensions {
					contnt := re.ReplaceAll(text, []byte(ext))
					data = append(data, []byte(contnt))
				}
			} else {
				text := reader.Text()

				// Always ignore comment lines starting with #
				text, ok = stripComments(text)
				if !ok {
					continue
				}
				
				// Only add if we haven't seen this payload before
				if !seen[text] {
					data = append(data, []byte(text))
					seen[text] = true
				}
				linesRead++
			}
		} else {
			text := reader.Text()

			// Always ignore comment lines starting with #
			text, ok = stripComments(text)
			if !ok {
				continue
			}
			
			// Only add original payload if not seen before
			if !seen[text] {
				data = append(data, []byte(text))
				seen[text] = true
			}
			linesRead++
			
			if w.keyword == "FUZZ" && len(w.config.Extensions) > 0 {
				for _, ext := range w.config.Extensions {
					// Remove dot from extension if present (for backward compatibility)
					cleanExt := strings.TrimPrefix(ext, ".")
					
					// Replace/add extension
					newPayload := replaceExtension(text, cleanExt)
					
					// Only add if we haven't seen this payload before
					if !seen[newPayload] {
						data = append(data, []byte(newPayload))
						seen[newPayload] = true
					}
				}
			}
		}
	}
	w.data = data
	return reader.Err()
}

// hasValidExtension checks if a string has a valid file extension (1-4 chars after last dot)
// Returns true if it has an extension, false otherwise
func hasValidExtension(text string) bool {
	// Find the last dot
	lastDot := strings.LastIndex(text, ".")
	if lastDot == -1 {
		return false
	}
	
	// Check if it's at the end or has invalid characters before
	if lastDot == len(text)-1 {
		return false
	}
	
	// Get the part after the last dot
	extension := text[lastDot+1:]
	
	// Extension should be 1-4 alphanumeric characters
	if len(extension) < 1 || len(extension) > 4 {
		return false
	}
	
	// Check if all characters are alphanumeric
	for _, char := range extension {
		if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9')) {
			return false
		}
	}
	
	return true
}

// removeExtension removes the file extension from a string if it has a valid one
func removeExtension(text string) string {
	if !hasValidExtension(text) {
		return text
	}
	
	lastDot := strings.LastIndex(text, ".")
	return text[:lastDot]
}

// replaceExtension replaces the extension of a file with a new one
// If no extension exists, it appends the new extension
func replaceExtension(text, newExt string) string {
	base := removeExtension(text)
	return base + "." + newExt
}

// stripComments removes all kind of comments and empty lines from the word
func stripComments(text string) (string, bool) {
	// Trim spaces from both ends
	trimmed := strings.TrimSpace(text)
	
	// If the line is empty after trimming, ignore it
	if trimmed == "" {
		return "", false
	}
	
	// If the line starts with a # ignoring any space on the left,
	// return blank.
	if strings.HasPrefix(strings.TrimLeft(text, " "), "#") {
		return "", false
	}

	// If the line has # later after a space, that's a comment.
	// Only send the word upto space to the routine.
	index := strings.Index(text, " #")
	if index == -1 {
		return text, true
	}
	return text[:index], true
}
