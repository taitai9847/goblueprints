package main

import (
	"bufio"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"

	"github.com/taitai9847/goblueprints/ch4/thesaurus"
)

func main() {
	// apiKey := os.Getenv("BHT_APIKEY")
	err := godotenv.Load("../../.env")
	if err != nil {
		fmt.Printf("読み込み出来ませんでした: %v", err)
	}
	apiKey := os.Getenv("API_KEY")
	fmt.Println("apiKey: ", apiKey)

	thesaurus := &thesaurus.BigHugh{APIKey: apiKey}
	s := bufio.NewScanner(os.Stdin)
	for s.Scan() {
		word := s.Text()
		syns, err := thesaurus.Synonyms(word)
		if err != nil {
			log.Fatalln("Failed when looking for synonyms for \""+word+"\"", err)
		}
		if len(syns) == 0 {
			log.Fatalln("Couldn't find any synonyms for \"" + word + "\"")
		}
		for _, syn := range syns {
			fmt.Println(syn)
		}
	}
}
