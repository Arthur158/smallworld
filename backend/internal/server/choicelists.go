package server

type ExtensionList struct {
	Name string
	Races []string
	Traits []string
}

type Extension struct {
	ExtensionName string      `json:"extensionName"`
	IsChecked     bool        `json:"isChecked"`
	RaceChoices   []ChoiceEntry `json:"raceChoices"`
	TraitChoices  []ChoiceEntry `json:"traitChoices"`
}

type ChoiceEntry struct {
    Choice string `json:"choice"`
    IsChecked bool `json:"isChecked"`
}

var extensions = []ExtensionList {
	{
		Name: "Base Game",
		Races : []string{"Amazons", "Elves", "Ghouls", "Giants", "Halflings", "Humans", "Orcs", "Ratmen", "Skeletons", "Sorcerers", "Tritons", "Trolls", "Wizards"},
		Traits: []string{"Alchemist", "Berserk", "Bivouacking", "Commando", "Diplomat", "Dragon Master", "Flying", "Forest", "Fortified", "Heroic", "Hill", "Merchant", "Pillaging", "Seafaring", "Spirit", "Stout", "Swamp", "Underworld", "Wealthy"},
	},
	{
		Name: "Cursed!",
		Races : []string{"Goblins", "Kobolds"},
		Traits: []string{"Hordes of", "Ransacking"},
	},
	{
		Name: "Sky Islands",
		Races : []string{"Drakons", "Khans", "Scarecrows", "Scavengers", "Wendigos"},
		Traits: []string{"Goldsmith", "Haggling", "Zeppelined", "Gunner"},
	},
	{
		Name: "Grand Dames",
		Races : []string{"Nomads", "Priestesses", "White Ladies"},
		Traits: []string{"Peace-loving"},
	},
	{
		Name: "Be Not Afraid",
		Races : []string{"Barbarians", "Leprechauns", "Witch Doctors", "Pixies"},
		Traits: []string{"Barricade", "Catapult", "Corrupt", "Imperial", "Mercenary"},
	},
	{
		Name: "Royal Bonus",
		Races : []string{"Fauns", "Shrubmen"},
		Traits: []string{"Aquatic", "Behemoth", "Fireball"},
	},
	{
		Name: "A Spider's Web",
		Races : []string{"Ice Witches"},
		Traits: []string{"Lava"},
	},
	{
		Name: "Underground",
		Races : []string{"Gnomes"},
		Traits: []string{},
	},
}
