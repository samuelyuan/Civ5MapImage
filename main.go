package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/samuelyuan/Civ5MapImage/fileio"
)

func main() {
	inputPtr := flag.String("input", "", "Input filename")
	outputPtr := flag.String("output", "output.png", "Output filename")
	modePtr := flag.String("mode", "physical", "Drawing mode")
	flag.Parse()

	fmt.Println("Input filename: ", *inputPtr)
	fmt.Println("Output filename: ", *outputPtr)
	fmt.Println("Mode: ", *modePtr)
	mapData, err := fileio.ReadCiv5MapFile(*inputPtr)
	if err != nil {
		log.Fatal("Failed to read input file: ", err)
	}

	mode := *modePtr
	if mode == "physical" {
		drawPhysicalMap(mapData, *outputPtr)
	} else if mode == "political" {
		drawPoliticalMap(mapData, *outputPtr)
	} else {
		log.Fatal("Invalid drawing mode: " + mode + ". Mode must be in this list [phyiscal, political].")
	}
}
