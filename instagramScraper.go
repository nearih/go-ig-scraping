package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/gocolly/colly"
)

// Define your target instagram account here
const instagramAccount string = `jane.bnk48official`

// Ajax paging URL
const nextPageURLTemplate string = `https://www.instagram.com/graphql/query/?query_hash=a5164aed103f24b03e7b7747a2d94e3c&variables={"id":"%s","first":12,"after":"%s"}`

// Structure - paging cursor
type pageInfo struct {
	EndCursor string `json:"end_cursor"`
	NextPage  bool   `json:"has_next_page"`
}

// Structure - first entry
type entryNode struct {
	EntryData struct {
		ProfilePage []struct {
			Graphql struct {
				User struct {
					ID    string `json:"id"`
					Media struct {
						Nodes []struct {
							ImageURL     string `json:"display_url"`
							ThumbnailURL string `json:"thumbnail_src"`
							IsVideo      bool   `json:"is_video"`
							Date         int    `json:"date"`
							Dimensions   struct {
								Width  int `json:"width"`
								Height int `json:"height"`
							} `json:"dimensions"`
							Likes struct {
								Count int `json:"count"`
							} `json:"edge_liked_by"`
						} `json:"node"`
						PageInfo pageInfo `json:"page_info"`
					} `json:"edge_owner_to_timeline_media"`
				} `json:"user"`
			} `json:"graphql"`
		} `json:"ProfilePage"`
	} `json:"entry_data"`
}

type entryData struct {
	EntryData string `json:"entry_data"`
}

type locale struct {
	Locale string `json:"locale"`
}

// Structure - next entry
type entryEdgeNode struct {
	Data struct {
		User struct {
			Container struct {
				PageInfo pageInfo `json:"page_info"`
				Edges    []struct {
					Node struct {
						ImageURL     string `json:"display_url"`
						ThumbnailURL string `json:"thumbnail_src"`
						IsVideo      bool   `json:"is_video"`
						Date         int    `json:"taken_at_timestamp"`
						Dimensions   struct {
							Width  int `json:"width"`
							Height int `json:"height"`
						}
						Likes struct {
							Count int `json:"count"`
						} `json:"edge_media_preview_like"`
					}
				} `json:"edges"`
			} `json:"edge_owner_to_timeline_media"`
		}
	} `json:"data"`
}

// Statistic Data - for csv file
var statData = [][]string{{"Image Name", "Likes"}}

func main() {

	var actualUserID string
	outputDir := fmt.Sprintf("./instagram_scrapped_image/")

	c := colly.NewCollector(
		colly.CacheDir("./_instagram_cache/"),
	)

	c.OnHTML("body > script:first-of-type", func(e *colly.HTMLElement) {
		jsonData := e.Text[strings.Index(e.Text, "{") : len(e.Text)-1]
		fmt.Println("")
		fmt.Println("")
		data := entryNode{}
		data2 := entryData{}
		locale := locale{}

		err := json.Unmarshal([]byte(jsonData), &data)
		if err != nil {

			log.Fatal("97	", err)
		}

		err = json.Unmarshal([]byte(jsonData), &data2)
		if err != nil {

			log.Fatal("98	", err)
		}
		err = json.Unmarshal([]byte(jsonData), &locale)
		if err != nil {

			log.Fatal("99	", err)
		}
		// jsonData = "[" + jsonData + "]"
		// jsonByte, err := json.Marshal(jsonData)
		// if err != nil {
		// 	log.Fatal("103	", err)
		// }
		fmt.Println("")
		fmt.Println("data	", data)
		fmt.Println("data2	", data2)
		fmt.Println("locale	", locale)
		fmt.Println("")
		// fmt.Println("jsonData	", jsonData)
		fmt.Println("")
		ioutil.WriteFile("jsonData.json", []byte(jsonData), 0644)
		log.Println("saving output to ", outputDir)
		err = os.MkdirAll(outputDir, os.ModePerm)
		if err != nil {
			fmt.Println("111	", err)
		}
		page := data.EntryData.ProfilePage[0]
		actualUserID = page.User.ID
		for _, obj := range page.User.Media.Nodes {
			if obj.IsVideo {
				continue
			}
			newStat := []string{filepath.Base(obj.ImageURL), fmt.Sprintf("%v", obj.Likes.Count)}
			statData = append(statData, newStat)
			c.Visit(obj.ImageURL)
		}
		if page.User.Media.PageInfo.NextPage {
			c.Visit(fmt.Sprintf(nextPageURLTemplate, actualUserID, page.User.Media.PageInfo.EndCursor))
		}
	})

	c.OnResponse(func(r *colly.Response) {
		if strings.Index(r.Headers.Get("Content-Type"), "image") > -1 {
			log.Println("Saving Image: " + r.FileName())
			err := r.Save(outputDir + r.FileName())
			if err != nil {
				fmt.Println("133	", err)
			}
			return
		}

		if strings.Index(r.Headers.Get("Content-Type"), "json") == -1 {
			return
		}

		data := entryEdgeNode{}
		err := json.Unmarshal(r.Body, &data)
		if err != nil {
			log.Fatal("145	", err)
		}

		fmt.Println("entry edge node	", data)
		for _, obj := range data.Data.User.Container.Edges {
			if obj.Node.IsVideo {
				continue
			}
			newStat := []string{filepath.Base(obj.Node.ImageURL), fmt.Sprintf("%v", obj.Node.Likes.Count)}
			statData = append(statData, newStat)
			err = c.Visit(obj.Node.ImageURL)
			if err != nil {
				fmt.Println("156	", err)
			}
		}
		if data.Data.User.Container.PageInfo.NextPage {
			err = c.Visit(fmt.Sprintf(nextPageURLTemplate, actualUserID, data.Data.User.Container.PageInfo.EndCursor))
			if err != nil {
				fmt.Println("162	", err)
			}
		} else {
			log.Println("Done Scraping")

			// Save CSV File -----------------------------
			file, err := os.Create("result.csv")
			checkError("Cannot create file", err)
			defer file.Close()

			writer := csv.NewWriter(file)
			defer writer.Flush()

			for _, value := range statData {
				err := writer.Write(value)
				checkError("Cannot write to file", err)
			}
		}
	})

	err := c.Visit("https://instagram.com/" + instagramAccount)
	if err != nil {
		fmt.Println("184	", err)
	}
}

func checkError(message string, err error) {
	if err != nil {
		log.Fatal(message, err)
	}
}
