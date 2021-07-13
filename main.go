/* onuwf 는 보드게임 "한밤의 늑대인간" 을 디스코드 봇으로 구현하는 프로젝트입니다. */

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	wfGame "onuwf.com/game"
	util "onuwf.com/util"

	"github.com/bwmarrin/discordgo"
)

const (
	prefix = "ㅁ"
)

var (
	isUserIn            map[string]bool
	uidToGameData       map[string]*wfGame.Game
	guildChanToGameData map[string]*wfGame.Game

	env map[string]string
	emj map[string]string
	rg  []wfGame.RoleGuide
)

func init() {
	env = EnvInit()
	emj = EmojiInit()
	RoleGuideInit(&rg)
	util.ReadJSON(rg)
	//util.MongoConn(env)

	isUserIn = make(map[string]bool)
	guildChanToGameData = make(map[string]*wfGame.Game)
	uidToGameData = make(map[string]*wfGame.Game)
}

func main() {
	dg, err := discordgo.New("Bot " + env["dgToken"])
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}
	dg.AddHandler(messageCreate)
	dg.AddHandler(messageReactionAdd)
	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc
	dg.Close()
}

func startgame(s *discordgo.Session, m *discordgo.MessageCreate) {
	enterUserIDChan := make(chan string, 1)
	quitUserIDChan := make(chan string)
	gameStartedChan := make(chan bool)
	curGame := wfGame.NewGame(m.GuildID, m.ChannelID, m.Author.ID, s, rg, emj, enterUserIDChan, quitUserIDChan, gameStartedChan)
	// Mutex 필요할 것으로 예상됨.
	curGame.UserList = append(curGame.UserList, wfGame.NewUser(m.Author.ID, "juhur", m.ChannelID, m.ChannelID))
	curGame.UserList = append(curGame.UserList, wfGame.NewUser(m.Author.ID, "min-jo", m.ChannelID, m.ChannelID))
	curGame.UserList = append(curGame.UserList, wfGame.NewUser(m.Author.ID, "kalee", m.ChannelID, m.ChannelID))
	guildChanToGameData[m.GuildID+m.ChannelID] = curGame
	uidToGameData[m.Author.ID] = curGame
	for {
		select {
		case curUID := <-curGame.EnterUserIDChan:
			isUserIn[curUID] = true
			guildChanToGameData[m.GuildID+curUID] = curGame
			uidToGameData[curUID] = curGame
		case curUID := <-curGame.QuitUserIDChan:
			delete(isUserIn, curUID)
			delete(uidToGameData, curUID)
		case <-curGame.GameStartedChan:
			return
		}
	}
}

// messageCreate() 입력한 메시지를 처리하는 함수
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}
	// 명령어모음
	if util.PrintHelpList(s, m, rg) {
		return
	}
	switch m.Content {
	case "ㅁ시작":
		if guildChanToGameData[m.GuildID+m.ChannelID] != nil {
			s.ChannelMessageSend(m.ChannelID, "게임을 진행중인 채널입니다.")
			return
		}
		if isUserIn[m.Author.ID] {
			s.ChannelMessageSend(m.ChannelID, "게임을 진행중인 사용자입니다.")
			return
		}
		isUserIn[m.Author.ID] = true
		go startgame(s, m)
	case "ㅁ강제종료":
		if isUserIn[m.Author.ID] {
			s.ChannelMessageSend(m.ChannelID, "3초 후 게임을 강제종료합니다.")
			time.Sleep(3 * time.Second)
			g := guildChanToGameData[m.GuildID+m.ChannelID]
			if m.Author.ID != g.MasterID {
				return
			}
			for _, user := range g.UserList {
				delete(isUserIn, user.UserID)
				delete(uidToGameData, user.UserID)
			}
			delete(guildChanToGameData, m.GuildID+m.ChannelID)
			g.CanFunc()
			s.ChannelMessageSend(m.ChannelID, "게임을 강제종료 했습니다.")
		}
	case "ㅁ투표":
		uidChan := make(chan string, 7)
		thisGame := wfGame.NewGame(m.GuildID, m.ChannelID, m.Author.ID, s, rg, emj, uidChan, nil, nil)
		thisGame.UserList = append(thisGame.UserList, wfGame.NewUser(m.Author.ID, "jae-kim", m.ChannelID, m.ChannelID))
		thisGame.UserList = append(thisGame.UserList, wfGame.NewUser(m.Author.ID, "juhur", m.ChannelID, m.ChannelID))
		thisGame.UserList = append(thisGame.UserList, wfGame.NewUser(m.Author.ID, "min-jo", m.ChannelID, m.ChannelID))
		thisGame.UserList = append(thisGame.UserList, wfGame.NewUser(m.Author.ID, "kalee", m.ChannelID, m.ChannelID))
		thisGame.UserList = append(thisGame.UserList, wfGame.NewUser(m.Author.ID, "apple", m.ChannelID, m.ChannelID))
		thisGame.UserList = append(thisGame.UserList, wfGame.NewUser(m.Author.ID, "banana", m.ChannelID, m.ChannelID))

		voted_list := make([]int, len(thisGame.UserList))
		temp := &wfGame.StateVote{thisGame, voted_list, len(thisGame.UserList), 0}
		guildChanToGameData[m.GuildID+m.ChannelID] = thisGame
		isUserIn[m.Author.ID] = true
		thisGame.CurState = temp
		wfGame.VoteProcess(s, thisGame)
	case "ㅁ확인":
		g := guildChanToGameData[m.GuildID+m.ChannelID]
		if g != nil {
			Server, _ := s.State.Guild(m.GuildID)
			Channel, _ := s.State.Channel(m.ChannelID)
			msg := "----------------------------------------------------\n"
			msg += "> 현재 서버: " + Server.Name + "\n"
			msg += "> 현재 채널: " + Channel.Name + "\n"
			msg += "> 현재 유저 수: " + strconv.Itoa(len(g.UserList)) + "\n"
			msg += "----------------------------------------------------\n"
			for i, user := range g.UserList {
				msg += "< " + strconv.Itoa(i+1) + "번 유저 `" + user.Nick() + "` >\n"
				msg += "원래직업: " + g.GetOriRole(user.UserID).String() + "\n"
				msg += "현재직업: " + g.GetRole(user.UserID).String() + "\n"
			}
			msg += "< 버려진 직업들 >\n"
			for i := 0; i < 3; i++ {
				msg += g.GetDisRole(i).String() + " "
			}
			msg += "\n"
			msg += "----------------------------------------------------\n"
			msg += "로그 메시지 :\n"
			for _, text := range g.LogMsg {
				msg += text + "\n"
			}
			msg += "----------------------------------------------------\n"
			s.ChannelMessageSend(m.ChannelID, msg)
		}
	}
}

// messageReactionAdd 함수는 인게임 버튼 이모지 상호작용 처리를 위한 이벤트 핸들러 함수입니다.
func messageReactionAdd(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
	//fmt.Println(r.UserID, r.MessageID, r.ChannelID, r.GuildID)
	// 봇 자기자신의 리액션 무시.
	if r.UserID == s.State.User.ID {
		return
	}
	// 게임 참가중이 아닌 사용자의 리액션 무시.
	// 단, 참가자가 아니면 참가 가능해야 함. 무시해버리면 참가 못 함.
	if !(isUserIn[r.UserID] || (!isUserIn[r.UserID] && r.Emoji.Name == emj["YES"])) {
		return
	}
	g := uidToGameData[r.UserID]
	if g == nil {
		g = guildChanToGameData[r.GuildID+r.ChannelID]
		if g == nil {
			return
		}
	}
	isUserIn[r.UserID] = true
	// 숫자 이모지 선택.
	for i := 1; i < 10; i++ {
		emjID := "n" + strconv.Itoa(i)
		if r.Emoji.Name == emj[emjID] {
			go g.CurState.PressNumBtn(s, r, i)
			break
		}
	}
	switch r.Emoji.Name {
	case emj["DISCARD"]:
		// 쓰레기통 이모지 선택.
		g.CurState.PressDisBtn(s, r)
	case emj["YES"]:
		// O 이모지 선택.
		g.CurState.PressYesBtn(s, r)
	case emj["NO"]:
		// X 이모지 선택.
		g.CurState.PressNoBtn(s, r)
	case emj["LEFT"]:
		// 왼쪽 화살표 선택.
		g.CurState.PressDirBtn(s, r, -1)
	case emj["RIGHT"]:
		// 오른쪽 화살표 선택.
		g.CurState.PressDirBtn(s, r, 1)
	}
}

// EnvInit 설치 환경 불러오기.
func EnvInit() map[string]string {
	envFile, err := os.Open("asset/env.json")
	if err != nil {
		log.Fatal(err)
	}
	defer envFile.Close()

	var byteValue []byte
	byteValue, err = ioutil.ReadAll(envFile)
	if err != nil {
		log.Fatal(err)
	}
	env := make(map[string]string)
	json.Unmarshal([]byte(byteValue), &env)
	return env
}

// RoleGuideInit 직업 가이드 에셋 불러오기.
func RoleGuideInit(rg *[]wfGame.RoleGuide) {
	rgFile, err := os.Open("asset/role_guide.json")
	if err != nil {
		log.Fatal(err)
	}
	defer rgFile.Close()

	var byteValue []byte
	byteValue, err = ioutil.ReadAll(rgFile)
	if err != nil {
		log.Fatal(err)
	}
	json.Unmarshal([]byte(byteValue), rg)
}

// EmojiInit 이모지 맵에 불러오기.
func EmojiInit() map[string]string {
	emjFile, err := os.Open("asset/emoji.json")
	if err != nil {
		log.Fatal(err)
	}
	defer emjFile.Close()

	var byteValue []byte
	byteValue, err = ioutil.ReadAll(emjFile)
	if err != nil {
		log.Fatal(err)
	}
	emj := make(map[string]string)
	json.Unmarshal([]byte(byteValue), &emj)
	return emj
}
