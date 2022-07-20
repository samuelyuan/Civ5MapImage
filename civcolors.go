package main

import (
	"image/color"
)

type CivColor struct {
	OuterColor color.RGBA
	InnerColor color.RGBA
}

func initCivColorMap() map[string]CivColor {
	civColorMap := make(map[string]CivColor)
	minorColor := color.RGBA{0, 0, 0, 255}
	civColorMap["PLAYERCOLOR_AMERICA"] = CivColor{
		OuterColor: color.RGBA{31, 51, 120, 255},   // blue
		InnerColor: color.RGBA{255, 255, 255, 255}, // white
	}
	civColorMap["PLAYERCOLOR_ARABIA"] = CivColor{
		OuterColor: color.RGBA{43, 87, 45, 255},  // dark green
		InnerColor: color.RGBA{146, 221, 9, 255}, // light green
	}
	civColorMap["PLAYERCOLOR_AUSTRIA"] = CivColor{
		OuterColor: color.RGBA{234, 0, 0, 255},     // red
		InnerColor: color.RGBA{255, 255, 255, 255}, // white
	}
	civColorMap["PLAYERCOLOR_AZTECS"] = CivColor{
		OuterColor: color.RGBA{161, 57, 34, 255},   // red
		InnerColor: color.RGBA{136, 238, 212, 255}, // light blue
	}
	civColorMap["PLAYERCOLOR_BRAZIL"] = CivColor{
		OuterColor: color.RGBA{149, 221, 10, 255}, // light green
		InnerColor: color.RGBA{41, 83, 44, 255},   // dark green
	}
	civColorMap["PLAYERCOLOR_EGYPT"] = CivColor{
		OuterColor: color.RGBA{255, 251, 3, 255}, // yellow
		InnerColor: color.RGBA{82, 0, 208, 255},  // purple
	}
	civColorMap["PLAYERCOLOR_CELTS"] = CivColor{
		OuterColor: color.RGBA{21, 91, 62, 255},    // dark green
		InnerColor: color.RGBA{147, 169, 255, 255}, // light blue
	}
	civColorMap["PLAYERCOLOR_CHINA"] = CivColor{
		OuterColor: color.RGBA{0, 148, 82, 255},    // green
		InnerColor: color.RGBA{255, 255, 255, 255}, // white
	}
	civColorMap["PLAYERCOLOR_ENGLAND"] = CivColor{
		OuterColor: color.RGBA{108, 2, 0, 255},     // dark red
		InnerColor: color.RGBA{255, 255, 255, 255}, // white
	}
	civColorMap["PLAYERCOLOR_ETHIOPIA"] = CivColor{
		OuterColor: color.RGBA{1, 39, 14, 255},   // dark green
		InnerColor: color.RGBA{255, 45, 45, 255}, // red
	}
	civColorMap["PLAYERCOLOR_FRANCE"] = CivColor{
		OuterColor: color.RGBA{65, 141, 253, 255},  // light blue
		InnerColor: color.RGBA{235, 235, 138, 255}, // white
	}
	civColorMap["PLAYERCOLOR_GERMANY"] = CivColor{
		OuterColor: color.RGBA{179, 177, 184, 255}, // gray
		InnerColor: color.RGBA{36, 43, 32, 255},    // dark gray
	}
	civColorMap["PLAYERCOLOR_GREECE"] = CivColor{
		OuterColor: color.RGBA{255, 255, 255, 255}, // white
		InnerColor: color.RGBA{65, 141, 253, 255},  // light blue
	}
	civColorMap["PLAYERCOLOR_HUNS"] = CivColor{
		OuterColor: color.RGBA{179, 177, 163, 255}, // gray
		InnerColor: color.RGBA{69, 0, 3, 255},      // dark red
	}
	civColorMap["PLAYERCOLOR_INCA"] = CivColor{
		OuterColor: color.RGBA{255, 184, 33, 255}, // yellow
		InnerColor: color.RGBA{6, 159, 119, 255},  // green
	}
	civColorMap["PLAYERCOLOR_INDIA"] = CivColor{
		OuterColor: color.RGBA{18, 135, 6, 255},   // green
		InnerColor: color.RGBA{255, 153, 49, 255}, // orange
	}
	civColorMap["PLAYERCOLOR_IROQUOIS"] = CivColor{
		OuterColor: color.RGBA{65, 86, 86, 255},    // gray
		InnerColor: color.RGBA{251, 201, 129, 255}, // beige
	}
	civColorMap["PLAYERCOLOR_JAPAN"] = CivColor{
		OuterColor: color.RGBA{255, 255, 255, 255}, // white
		InnerColor: color.RGBA{184, 0, 0, 255},     // red
	}
	civColorMap["PLAYERCOLOR_MAYA"] = CivColor{
		OuterColor: color.RGBA{197, 140, 98, 255}, // yellow
		InnerColor: color.RGBA{23, 62, 65, 255},   // dark blue
	}
	civColorMap["PLAYERCOLOR_MINOR_BLUE"] = CivColor{
		OuterColor: minorColor,
		InnerColor: color.RGBA{0, 0, 255, 255},
	}
	civColorMap["PLAYERCOLOR_MINOR_CYAN"] = CivColor{
		OuterColor: minorColor,
		InnerColor: color.RGBA{0, 255, 255, 255},
	}
	civColorMap["PLAYERCOLOR_MINOR_GOLDENROD"] = CivColor{
		OuterColor: minorColor,
		InnerColor: color.RGBA{222, 159, 0, 255},
	}
	civColorMap["PLAYERCOLOR_MINOR_GREEN"] = CivColor{
		OuterColor: minorColor,
		InnerColor: color.RGBA{0, 255, 0, 255},
	}
	civColorMap["PLAYERCOLOR_MINOR_GRAY"] = CivColor{
		OuterColor: minorColor,
		InnerColor: color.RGBA{178, 178, 178, 255},
	}
	civColorMap["PLAYERCOLOR_MINOR_LIGHT_BLUE"] = CivColor{
		OuterColor: minorColor,
		InnerColor: color.RGBA{127, 178, 255, 255},
	}
	civColorMap["PLAYERCOLOR_MINOR_LIGHT_GREEN"] = CivColor{
		OuterColor: minorColor,
		InnerColor: color.RGBA{127, 255, 127, 255},
	}
	civColorMap["PLAYERCOLOR_MINOR_LIGHT_ORANGE"] = CivColor{
		OuterColor: minorColor,
		InnerColor: color.RGBA{229, 166, 82, 255},
	}
	civColorMap["PLAYERCOLOR_MINOR_LIGHT_PURPLE"] = CivColor{
		OuterColor: minorColor,
		InnerColor: color.RGBA{178, 153, 255, 255},
	}
	civColorMap["PLAYERCOLOR_MINOR_LIGHT_YELLOW"] = CivColor{
		OuterColor: minorColor,
		InnerColor: color.RGBA{255, 255, 127, 255},
	}
	civColorMap["PLAYERCOLOR_MINOR_PEACH"] = CivColor{
		OuterColor: minorColor,
		InnerColor: color.RGBA{255, 217, 143, 255},
	}
	civColorMap["PLAYERCOLOR_MINOR_MIDDLE_CYAN"] = CivColor{
		OuterColor: minorColor,
		InnerColor: color.RGBA{0, 163, 181, 255},
	}
	civColorMap["PLAYERCOLOR_MINOR_WHITE"] = CivColor{
		OuterColor: minorColor,
		InnerColor: color.RGBA{255, 255, 255, 255},
	}
	civColorMap["PLAYERCOLOR_MONGOL"] = CivColor{
		OuterColor: color.RGBA{81, 0, 8, 255},    // dark red
		InnerColor: color.RGBA{255, 120, 0, 255}, // orange
	}
	civColorMap["PLAYERCOLOR_MOROCCO"] = CivColor{
		OuterColor: color.RGBA{144, 2, 0, 255},   // dark red
		InnerColor: color.RGBA{39, 178, 79, 255}, // green
	}
	civColorMap["PLAYERCOLOR_NETHERLANDS"] = CivColor{
		OuterColor: color.RGBA{255, 143, 0, 255},   // orange
		InnerColor: color.RGBA{255, 255, 255, 255}, // white
	}
	civColorMap["PLAYERCOLOR_OTTOMAN"] = CivColor{
		OuterColor: color.RGBA{247, 248, 199, 255}, // white
		InnerColor: color.RGBA{18, 82, 30, 255},    // green
	}
	civColorMap["PLAYERCOLOR_PERSIA"] = CivColor{
		OuterColor: color.RGBA{176, 7, 3, 255},    // red
		InnerColor: color.RGBA{245, 230, 55, 255}, // yellow
	}
	civColorMap["PLAYERCOLOR_PORTUGAL"] = CivColor{
		OuterColor: color.RGBA{255, 255, 255, 255}, // white
		InnerColor: color.RGBA{3, 20, 124, 255},    // dark blue
	}
	civColorMap["PLAYERCOLOR_ROME"] = CivColor{
		OuterColor: color.RGBA{70, 0, 118, 255},  // purple
		InnerColor: color.RGBA{239, 198, 0, 255}, // yellow
	}
	civColorMap["PLAYERCOLOR_RUSSIA"] = CivColor{
		OuterColor: color.RGBA{238, 180, 0, 255}, // yellow
		InnerColor: color.RGBA{0, 0, 0, 255},     // black
	}
	civColorMap["PLAYERCOLOR_SIAM"] = CivColor{
		OuterColor: color.RGBA{245, 230, 55, 255}, // yellow
		InnerColor: color.RGBA{176, 7, 3, 255},    // red
	}
	civColorMap["PLAYERCOLOR_SPAIN"] = CivColor{
		OuterColor: color.RGBA{83, 26, 26, 255},    // dark red
		InnerColor: color.RGBA{244, 168, 168, 255}, // pink
	}
	civColorMap["PLAYERCOLOR_SWEDEN"] = CivColor{
		OuterColor: color.RGBA{7, 7, 165, 255},   // dark blue
		InnerColor: color.RGBA{248, 246, 2, 255}, // yellow
	}
	civColorMap["PLAYERCOLOR_WHITE"] = CivColor{
		OuterColor: color.RGBA{219, 5, 5, 255},     // light red
		InnerColor: color.RGBA{229, 229, 229, 255}, // white
	}
	civColorMap["PLAYERCOLOR_ZULU"] = CivColor{
		OuterColor: color.RGBA{255, 231, 213, 255}, // beige
		InnerColor: color.RGBA{106, 49, 24, 255},   // dark red
	}
	return civColorMap
}
