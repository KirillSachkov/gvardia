package ui

// cyrillicToLatin maps ЙЦУКЕН Cyrillic runes to the Latin key at the same
// physical position on a US QWERTY keyboard. Bubble Tea reports the typed
// character (e.g. "в" for the physical D key), and the US-layout base code is
// only available under the Kitty protocol, so gvardia normalizes here instead.
var cyrillicToLatin = map[string]string{
	// top letter row
	"й": "q", "ц": "w", "у": "e", "к": "r", "е": "t", "н": "y", "г": "u",
	"ш": "i", "щ": "o", "з": "p", "х": "[", "ъ": "]",
	// home letter row
	"ф": "a", "ы": "s", "в": "d", "а": "f", "п": "g", "р": "h", "о": "j",
	"л": "k", "д": "l", "ж": ";", "э": "'",
	// bottom letter row
	"я": "z", "ч": "x", "с": "c", "м": "v", "и": "b", "т": "n", "ь": "m",
	"б": ",", "ю": ".", "ё": "`",
	// uppercase (shift): map to the uppercase Latin so bound keys like R/A/X/C fire
	"Й": "Q", "Ц": "W", "У": "E", "К": "R", "Е": "T", "Н": "Y", "Г": "U",
	"Ш": "I", "Щ": "O", "З": "P",
	"Ф": "A", "Ы": "S", "В": "D", "А": "F", "П": "G", "Р": "H", "О": "J",
	"Л": "K", "Д": "L",
	"Я": "Z", "Ч": "X", "С": "C", "М": "V", "И": "B", "Т": "N", "Ь": "M",
}

// normalizeKey rewrites a single Cyrillic key to its US-QWERTY Latin equivalent
// so keybinds work under a Russian layout. Multi-character key names
// (enter, esc, tab, arrows, ctrl+c) and already-Latin keys pass through unchanged.
func normalizeKey(s string) string {
	if v, ok := cyrillicToLatin[s]; ok {
		return v
	}
	return s
}
