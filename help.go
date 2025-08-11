package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ffuf/ffuf/v2/pkg/ffuf"
)

type UsageSection struct {
	Name          string
	Description   string
	Flags         []UsageFlag
	Hidden        bool
	ExpectedFlags []string
}

// PrintSection prints out the section name, description and each of the flags
func (u *UsageSection) PrintSection(max_length int, extended bool) {
	// Do not print if extended usage not requested and section marked as hidden
	if !extended && u.Hidden {
		return
	}
	fmt.Printf("%s:\n", u.Name)
	for _, f := range u.Flags {
		f.PrintFlag(max_length)
	}
	fmt.Printf("\n")
}

type UsageFlag struct {
	Name        string
	Description string
	Default     string
}

// PrintFlag prints out the flag name, usage string and default value
func (f *UsageFlag) PrintFlag(max_length int) {
	// Create format string, used for padding
	format := fmt.Sprintf("  -%%-%ds %%s", max_length)
	if f.Default != "" {
		format = format + " (default: %s)\n"
		fmt.Printf(format, f.Name, f.Description, f.Default)
	} else {
		format = format + "\n"
		fmt.Printf(format, f.Name, f.Description)
	}
}

func Usage() {
	u_http := UsageSection{
		Name:          "HTTP OPTIONS",
		Description:   "Options controlling the HTTP request and its parts.",
		Flags:         make([]UsageFlag, 0),
		Hidden:        false,
		ExpectedFlags: []string{"cc", "ck", "H", "X", "b", "d", "r", "u", "raw", "recursion", "recursion-depth", "recursion-strategy", "replay-proxy", "timeout", "ignore-body", "x", "sni", "http2"},
	}
	u_general := UsageSection{
		Name:          "GENERAL OPTIONS",
		Description:   "",
		Flags:         make([]UsageFlag, 0),
		Hidden:        false,
		ExpectedFlags: []string{"ac", "acc", "ack", "ach", "acs", "aiuto", "c", "config", "debug-req", "json", "maxtime", "maxtime-job", "noninteractive", "p", "rate", "scraperfile", "scrapers", "search", "s", "sa", "se", "sf", "t", "v", "V"},
	}
	u_compat := UsageSection{
		Name:          "COMPATIBILITY OPTIONS",
		Description:   "Options to ensure compatibility with other pieces of software.",
		Flags:         make([]UsageFlag, 0),
		Hidden:        true,
		ExpectedFlags: []string{"compressed", "cookie", "data", "data-ascii", "data-binary", "i", "k"},
	}
	u_matcher := UsageSection{
		Name:          "MATCHER OPTIONS",
		Description:   "Matchers for the response filtering.",
		Flags:         make([]UsageFlag, 0),
		Hidden:        false,
		ExpectedFlags: []string{"mmode", "mc", "ml", "mr", "ms", "mt", "mw"},
	}
	u_filter := UsageSection{
		Name:          "FILTER OPTIONS",
		Description:   "Filters for the response filtering.",
		Flags:         make([]UsageFlag, 0),
		Hidden:        false,
		ExpectedFlags: []string{"fmode", "fc", "fl", "fr", "fs", "ft", "fw"},
	}
	u_input := UsageSection{
		Name:          "INPUT OPTIONS",
		Description:   "Options for input data for fuzzing. Wordlists and input generators.",
		Flags:         make([]UsageFlag, 0),
		Hidden:        false,
		ExpectedFlags: []string{"D", "enc", "ic", "input-cmd", "input-num", "input-shell", "l", "mode", "request", "request-proto", "S", "vhost", "vhost-domain", "e", "w"},
	}
	u_output := UsageSection{
		Name:          "OUTPUT OPTIONS",
		Description:   "Options for output. Output file formats, file names and debug file locations.",
		Flags:         make([]UsageFlag, 0),
		Hidden:        false,
		ExpectedFlags: []string{"audit-log", "debug-log", "o", "of", "od", "or"},
	}
	sections := []UsageSection{u_http, u_general, u_compat, u_matcher, u_filter, u_input, u_output}

	// Populate the flag sections
	max_length := 0
	flag.VisitAll(func(f *flag.Flag) {
		found := false
		for i, section := range sections {
			if ffuf.StrInSlice(f.Name, section.ExpectedFlags) {
				sections[i].Flags = append(sections[i].Flags, UsageFlag{
					Name:        f.Name,
					Description: f.Usage,
					Default:     f.DefValue,
				})
				found = true
			}
		}
		if !found {
			fmt.Printf("DEBUG: Flag %s was found but not defined in help.go.\n", f.Name)
			os.Exit(1)
		}
		if len(f.Name) > max_length {
			max_length = len(f.Name)
		}
	})

	fmt.Printf("FUFFA - FFUF Using Fantastic Formats And colors - v%s\n\n", ffuf.Version())

	// Print out the sections
	for _, section := range sections {
		section.PrintSection(max_length, false)
	}

	// Usage examples.
	fmt.Printf("EXAMPLE USAGE:\n")

	fmt.Printf("  Fuzz file paths from wordlist.txt, match all responses but filter out those with content-size 42.\n")
	fmt.Printf("  Colored, verbose output.\n")
	fmt.Printf("    fuffa -w wordlist.txt -u https://example.org/FUZZ -mc all -fs 42 -c -v\n\n")

	fmt.Printf("  Fuzz Host-header, match HTTP 200 responses.\n")
	fmt.Printf("    fuffa -w hosts.txt -u https://example.org/ -H \"Host: FUZZ\" -mc 200\n\n")

	fmt.Printf("  Fuzz POST JSON data. Match all responses not containing text \"error\".\n")
	fmt.Printf("    fuffa -w entries.txt -u https://example.org/ -X POST -H \"Content-Type: application/json\" \\\n")
	fmt.Printf("      -d '{\"name\": \"FUZZ\", \"anotherkey\": \"anothervalue\"}' -fr \"error\"\n\n")

	fmt.Printf("  Fuzz multiple locations. Match only responses reflecting the value of \"VAL\" keyword. Colored.\n")
	fmt.Printf("    fuffa -w params.txt:PARAM -w values.txt:VAL -u https://example.org/?PARAM=VAL -mr \"VAL\" -c\n\n")

	fmt.Printf("  More information and examples: https://github.com/ffuf/ffuf\n\n")
}

func ItalianUsage() {
	fmt.Printf("FUFFA - FFUF Using Fantastic Formats And colors - v%s\n\n", ffuf.Version())
	
	fmt.Printf("OPZIONI HTTP:\n")
	fmt.Printf("  -H                  Header HTTP `\"Nome: Valore\"`, separato da due punti. Sono accettate più flag -H.\n")
	fmt.Printf("  -X                  Metodo HTTP da utilizzare\n")
	fmt.Printf("  -d                  Dati POST\n")
	fmt.Printf("  -r                  Segui redirect (default: false)\n")
	fmt.Printf("  -u                  URL di destinazione\n")
	fmt.Printf("  -timeout            Timeout per le richieste HTTP in secondi. (default: 10)\n")
	fmt.Printf("  -x                  URL Proxy (SOCKS5 o HTTP). Esempio: http://127.0.0.1:8080\n")
	fmt.Printf("  -recursion          Scansiona ricorsivamente. Solo la keyword FUZZ è supportata. (default: false)\n")
	fmt.Printf("  -recursion-depth    Profondità massima di ricorsione. (default: 0)\n")
	fmt.Printf("\n")
	
	fmt.Printf("OPZIONI MATCHER:\n")
	fmt.Printf("  -mc                 Codici di stato HTTP da includere, o \"all\" per tutto. (default: 200-299,301,302,307,401,403,405,500)\n")
	fmt.Printf("  -ml                 Numero di righe da includere nella risposta\n")
	fmt.Printf("  -mr                 Regex da includere\n")
	fmt.Printf("  -ms                 Dimensione della risposta HTTP da includere\n")
	fmt.Printf("  -mw                 Numero di parole da includere nella risposta\n")
	fmt.Printf("\n")
	
	fmt.Printf("OPZIONI FILTER:\n")
	fmt.Printf("  -fc                 Codici di stato HTTP da escludere. Lista separata da virgole\n")
	fmt.Printf("  -fl                 Numero di righe da escludere nella risposta. Lista separata da virgole\n")
	fmt.Printf("  -fr                 Regex da escludere\n")
	fmt.Printf("  -fs                 Dimensione della risposta HTTP da escludere. Lista separata da virgole\n")
	fmt.Printf("  -fw                 Numero di parole da escludere nella risposta. Lista separata da virgole\n")
	fmt.Printf("\n")
	
	fmt.Printf("OPZIONI COMUNI:\n")
	fmt.Printf("  -w                  Percorso del file wordlist e (opzionale) keyword separata da due punti. es. '/path/to/wordlist:KEYWORD'\n")
	fmt.Printf("  -t                  Numero di thread concorrenti. (default: 40)\n")
	fmt.Printf("  -s                  Non stampare informazioni aggiuntive (modalità silenziosa) (default: false)\n")
	fmt.Printf("  -o                  Scrivi output su file\n")
	fmt.Printf("  -v                  Output verboso, stampa URL completo e posizione di redirect (se presente) con i risultati. (default: false)\n")
	fmt.Printf("  -h, --help          Mostra l'help completo con tutte le opzioni disponibili (in inglese).\n")
	fmt.Printf("\n")
	
	fmt.Printf("ESEMPIO D'USO:\n")
	fmt.Printf("  Ricerca directory da wordlist.txt, includi tutte le risposte ma escludi quelle con dimensione 42.\n")
	fmt.Printf("    fuffa -w wordlist.txt -u https://example.org/FUZZ -mc all -fs 42\n\n")
	
	fmt.Printf("  Fuzz header Host, includi solo risposte HTTP 200.\n")
	fmt.Printf("    fuffa -w hosts.txt -u https://example.org/ -H \"Host: FUZZ\" -mc 200\n\n")
	
	fmt.Printf("  Per maggiori informazioni ed esempi: https://github.com/ffuf/ffuf\n\n")
}
