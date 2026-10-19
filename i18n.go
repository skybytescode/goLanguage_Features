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
	OfflineHeading    string
	OfflineLead       string
	OfflineButton     string
	OfflineProgress   string // %d din %d
	OfflineDone       string
	OfflineFailed     string // %d pagini nesalvate
}

var ro = Strings{
	HomeTitle:         "Go, de la primul program până în runtime",
	HomeLead:          "Fiecare subiect are 4 niveluri. Toate nivelurile acoperă aceleași funcționalități ale limbajului; diferă doar cât de adânc intrăm în ele.",
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
	OfflineHeading:    "Aplicație și offline",
	OfflineLead:       "Pe iPhone: Safari → Share → „Add to Home Screen”. Pe Android și desktop: meniul browserului → „Instalează aplicația”. Paginile vizitate rămân disponibile fără internet; poți salva și toate lecțiile odată.",
	OfflineButton:     "Salvează toate lecțiile pentru offline",
	OfflineProgress:   "Se salvează… %d din %d",
	OfflineDone:       "Gata: toate lecțiile sunt disponibile offline.",
	OfflineFailed:     "Salvare incompletă: %d pagini nu au putut fi descărcate. Încearcă din nou.",
}

var en = Strings{
	HomeTitle:         "Go, from your first program to the runtime",
	HomeLead:          "Every topic has 4 levels. All levels cover the same language features; they differ only in how deep we go.",
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
	OfflineHeading:    "App and offline",
	OfflineLead:       "On iPhone: Safari → Share → “Add to Home Screen”. On Android and desktop: the browser menu → “Install app”. Visited pages stay available without internet; you can also save every lesson at once.",
	OfflineButton:     "Save all lessons for offline",
	OfflineProgress:   "Saving… %d of %d",
	OfflineDone:       "Done: every lesson is available offline.",
	OfflineFailed:     "Incomplete: %d pages couldn't be downloaded. Try again.",
}

func strs(lang Lang) *Strings {
	if lang == En {
		return &en
	}
	return &ro
}
