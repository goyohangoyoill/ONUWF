/* "ㅁ참고" 명령어 관련 함수 */
package util

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"

	wfGame "onuwf.com/game"
)

var (
	noteTitle string
	noteMsg   string
)

type note struct {
	Title string     `json:"title"`
	Line  []noteLine `json:"line"`
}

type noteLine struct {
	Bold string `json:"bold"`
	Post string `json:"post"`
}

// note.json 파일 읽어서 "ㅁ참고" 실행시 출력할 데이터 세팅
func readNoteJSON(rg []wfGame.RoleGuide) {
	noteFile, err := os.Open("./asset/note.json")
	if err != nil {
		log.Fatal(err)
		return
	}
	defer noteFile.Close()
	var note note
	byteValue, err := ioutil.ReadAll(noteFile)
	if err != nil {
		log.Fatal(err)
		return
	}
	json.Unmarshal(byteValue, &note)

	noteTitle = "**" + note.Title + "**"
	noteMsg = ""
	for i := 0; i < len(note.Line); i++ {
		if len(note.Line[i].Bold) > 0 {
			noteMsg += "**" + note.Line[i].Bold + "**"
		}
		noteMsg += note.Line[i].Post + "\n"
	}
	list := roleList(rg)
	for i, item := range list {
		noteMsg += item + " "
		if i%4 == 3 {
			noteMsg += "\n"
		}
	}
}