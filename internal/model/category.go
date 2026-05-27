package model

type Category string

const (
	CategoryTurf    = "turf"
	CategoryGym     = "gym"
	CategoryCoach   = "coach"
	CategoryClass   = "class"
	CategoryFeature = "feature"
)

var DisplayNames = map[string]string{
	"turf":    "Playing Turfs",
	"gym":     "Gyms",
	"coach":   "Coaches",
	"class":   "Classes",
	"feature": "App Features",
}

var allCategories = []string{
	CategoryTurf,
	CategoryGym,
	CategoryCoach,
	CategoryClass,
	CategoryFeature,
}

func IsValid(c string) bool {
	_, ok := DisplayNames[c]
	return ok
}

func All() []string {
	out := make([]string, len(allCategories))
	copy(out, allCategories)
	return out
}
