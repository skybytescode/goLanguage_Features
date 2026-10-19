package main

// Textele interfeței, în fiecare limbă.

type Strings struct {
	HomeTitle         string
	HomeLead          string
	LevelsHeading     string
	TopicsHeading     string
	SearchPlaceholder string
	SearchLabel       string
	SwitchLabel       string
	BadgeCompileFail  string
	BadgePanic        string
	Run               string
	Copy              string
	Copied            string
	NoResults         string // %s e căutarea
	OneResult         string
	ManyResults       string // %d e numărul de rezultate
	ManyResultsDe     string // română: „20 de rezultate”
}

var ro = Strings{
	HomeTitle:         "Go, de la primul program până în runtime",
	HomeLead:          "Fiecare subiect are 3 niveluri: Junior, Middle și Senior. Toate acoperă aceleași funcționalități ale limbajului; diferă doar cât de adânc intrăm în ele.",
	LevelsHeading:     "Niveluri",
	TopicsHeading:     "Subiecte",
	SearchPlaceholder: "Caută…  ( / )",
	SearchLabel:       "Caută în tot conținutul",
	SwitchLabel:       "Switch to English",
	BadgeCompileFail:  "Nu compilează (intenționat)",
	BadgePanic:        "Eșuează la rulare (intenționat)",
	Run:               "▶ Rulează",
	Copy:              "Copiază",
	Copied:            "Copiat",
	NoResults:         "Niciun rezultat pentru „%s”",
	OneResult:         "1 rezultat",
	ManyResults:       "%d rezultate",
	ManyResultsDe:     "%d de rezultate",
}

var en = Strings{
	HomeTitle:         "Go, from your first program to the runtime",
	HomeLead:          "Every topic has 3 levels: Junior, Middle and Senior. All levels cover the same language features; they differ only in how deep we go.",
	LevelsHeading:     "Levels",
	TopicsHeading:     "Topics",
	SearchPlaceholder: "Search…  ( / )",
	SearchLabel:       "Search all content",
	SwitchLabel:       "Treci la română",
	BadgeCompileFail:  "Does not compile (on purpose)",
	BadgePanic:        "Fails at run time (on purpose)",
	Run:               "▶ Run",
	Copy:              "Copy",
	Copied:            "Copied",
	NoResults:         "No results for “%s”",
	OneResult:         "1 result",
	ManyResults:       "%d results",
	ManyResultsDe:     "%d results",
}

func strs(lang Lang) *Strings {
	if lang == En {
		return &en
	}
	return &ro
}
