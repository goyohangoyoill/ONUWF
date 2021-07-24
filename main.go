/* onuwf 는 보드게임 "한밤의 늑대인간" 을 디스코드 봇으로 구현하는 프로젝트입니다. */

package main

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	embed "github.com/clinet/discordgo-embed"
	wfGame "github.com/goyohangoyoill/ONUWF/game"
	util "github.com/goyohangoyoill/ONUWF/util"
	json "github.com/goyohangoyoill/ONUWF/util/json"

	"github.com/bwmarrin/discordgo"
)

var (
	isUserIn            map[string]bool
	uidToGameData       map[string]*wfGame.Game
	guildChanToGameData map[string]*wfGame.Game
	fqChanMap           map[string]chan bool
	env                 map[string]string
	emj                 map[string]string
	rg                  []json.RoleGuide
	config              json.Config
	isNickChange        map[string]bool
	chNick              map[string]chan string
)

/*
type LoadDBInfo struct {
	MatchedUserList []*wfGame.User
	LastRoleSeq     []wfGame.Role //User로

}/
type SaveDBInfo struct {
	CurUserList []*wfGame.User
	CurRoleSeq  []int
	mUserID     string
}
*/
func init() {
	env = json.EnvInit()
	emj = json.EmojiInit()
	json.RoleGuideInit(&rg)
	config = json.ReadConfigJson()
	json.ReadJSON(rg, config.Prefix)
	//util.MongoConn(env)

	isUserIn = make(map[string]bool)
	guildChanToGameData = make(map[string]*wfGame.Game)
	uidToGameData = make(map[string]*wfGame.Game)
	fqChanMap = make(map[string]chan bool)
	isNickChange = make(map[string]bool)
	chNick = make(map[string]chan string)
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
	fqChanMap[m.GuildID+m.ChannelID] = make(chan bool, 1)
	curGame := wfGame.NewGame(m.GuildID, m.ChannelID, m.Author.ID, s, rg, emj, config, enterUserIDChan, quitUserIDChan, gameStartedChan, env)
	// Mutex 필요할 것으로 예상됨.
	guildChanToGameData[m.GuildID+m.ChannelID] = curGame
	uidToGameData[m.Author.ID] = curGame
	flag := false
	// juhur comment out
	for {
		if flag {
			break
		}
		select {
		case curUID := <-curGame.EnterUserIDChan:
			isUserIn[curUID] = true
			guildChanToGameData[m.GuildID+curUID] = curGame
			uidToGameData[curUID] = curGame
			// juhur comment out
		case curUID := <-curGame.QuitUserIDChan:
			delete(isUserIn, curUID)
			delete(uidToGameData, curUID)
		case _ = <-curGame.GameStartedChan:
			flag = true
			// juhur comment out
			SaveStartDB(curGame)
		}
	}
	<-curGame.GameStartedChan
	fqChanMap[m.GuildID+m.ChannelID] <- true
	g := guildChanToGameData[m.GuildID+m.ChannelID]
	if g == nil {
		<-fqChanMap[m.GuildID+m.ChannelID]
		return
	}
	// 여기에 DB 갱신 넣으면 됨.
	SaveEndDB(curGame)
	for _, user := range g.UserList {
		delete(isUserIn, user.UserID)
		delete(uidToGameData, user.UserID)
	}
	delete(guildChanToGameData, m.GuildID+m.ChannelID)
	g.CanFunc()
	s.ChannelMessageSend(m.ChannelID, "게임이 종료 되었습니다.")
	<-fqChanMap[m.GuildID+m.ChannelID]
}

// 게임 시작 시 save (user nick, lastrole 정보 저장)
func SaveStartDB(g *wfGame.Game) {
	conn, ctx := util.MongoConn(env)
	rLen := len(g.RoleView)
	RoleID := make([]int, rLen)
	// 게임 시작 시 설정 직업 정보를 가진 배열 초기
	for i := 0; i < rLen; i++ {
		RoleID[i] = g.RoleView[i].ID()
	}
	UserInfo := make([]*util.UserData, 0)
	uLen := len(g.UserList)
	for i := 0; i < uLen; i++ {
		UserInfo = append(UserInfo, &util.UserData{g.UserList[i].UserID, g.UserList[i].Nick(), "", time.Time{}, 0, 0, nil, nil})
	}
	sDB := util.SaveDBInfo{UserInfo, RoleID, g.MasterID}
	util.SetStartUser(sDB, "User", conn.Database("ONUWF"), ctx)
}

func SaveUserInit(g *wfGame.Game) []util.User {
	uLen := len(g.UserList)
	users := make([]util.User, 0, uLen)
	win := false
	for i := 0; i < uLen; i++ {
		user := util.User{}
		user.UID = g.UserList[i].UserID
		user.Nick = g.UserList[i].Nick()
		user.OriRole = g.GetOriRole(g.UserList[i].UserID).String()
		user.LastRole = g.GetRole(g.UserList[i].UserID).String()
		if (g.GetRole(g.UserList[i].UserID).String() == (&wfGame.Werewolf{}).String()) || (g.GetRole(g.UserList[i].UserID).String() == (&wfGame.Minion{}).String()) {
			win = g.WerewolfTeamWin
		} else if (g.GetRole(g.UserList[i].UserID).String()) == (&wfGame.Tanner{}).String() {
			win = g.TannerTeamWin
		} else {
			win = g.VillagerTeamWin
		}
		user.IsWin = win
		users = append(users, user)
	}
	return users
}

func SaveGameInit(g *wfGame.Game) util.GameData {
	sGame := util.GameData{}
	sGame.GuildID = g.GuildID
	sGame.ChanID = g.ChanID
	sGame.MasterID = g.MasterID
	RoleList := make([]string, 0, len(g.RoleView))
	for i := 0; i < len(g.RoleView); i++ {
		RoleList = append(RoleList, g.RoleView[i].String())
	}
	sGame.RoleList = RoleList
	sGame.UserList = SaveUserInit(g)
	disRole := make([]string, 0, len(g.DisRole))
	for i := 0; i < len(g.DisRole); i++ {
		disRole = append(disRole, g.DisRole[i].String())
	}
	sGame.OriDisRole = disRole
	sGame.LastDisRole = disRole

	return sGame
}

func SaveEndDB(g *wfGame.Game) {
	conn, ctx := util.MongoConn(env)
	sGame := SaveGameInit(g)
	t := time.Now()
	curGameOID := util.SaveGame(sGame, t, "Game", conn.Database("ONUWF"), ctx)
	uLen := len(g.UserList)
	win := false
	for i := 0; i < uLen; i++ {
		if (g.GetRole(g.UserList[i].UserID).String() == (&wfGame.Werewolf{}).String()) || (g.GetRole(g.UserList[i].UserID).String() == (&wfGame.Minion{}).String()) {
			win = g.WerewolfTeamWin
		} else if (g.GetRole(g.UserList[i].UserID).String()) == (&wfGame.Tanner{}).String() {
			win = g.TannerTeamWin
		} else {
			win = g.VillagerTeamWin
		}
		lUser, _ := util.LoadEachUser(g.UserList[i].UserID, true, "User", conn.Database("ONUWF"), ctx)
		util.SaveEachUser(&lUser, curGameOID, win, t, "User", conn.Database("ONUWF"), ctx)
	}
}

// messageCreate() 입력한 메시지를 처리하는 함수
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}
	// 명령어모음
	if json.PrintHelpList(s, m, rg, config.Prefix) {
		return
	}
	if isNickChange[m.Author.ID] {
		chNick[m.Author.ID] <- m.Content
		return
	}
	switch m.Content {
	case config.Prefix + "시작":
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
	case config.Prefix + "강제종료":
		if isUserIn[m.Author.ID] {
			curChan := fqChanMap[m.GuildID+m.ChannelID]
			// Mutex Lock
			curChan <- true
			g := guildChanToGameData[m.GuildID+m.ChannelID]
			if g == nil {
				<-curChan
				return
			}
			if m.Author.ID != g.MasterID {
				<-curChan
				return
			}
			s.ChannelMessageSend(m.ChannelID, "3초 후 게임을 강제종료합니다.")
			time.Sleep(3 * time.Second)
			g = guildChanToGameData[m.GuildID+m.ChannelID]
			if g == nil {
				<-curChan
			}
			for _, user := range g.UserList {
				delete(isUserIn, user.UserID)
				delete(uidToGameData, user.UserID)
			}
			delete(guildChanToGameData, m.GuildID+m.ChannelID)
			g.CanFunc()
			s.ChannelMessageSend(m.ChannelID, "게임을 강제종료 했습니다.")
			// Mutex Release
			<-curChan
		}
	case config.Prefix + "관전":
		g := guildChanToGameData[m.GuildID+m.ChannelID]
		if g == nil {
			return
		}
		if len(g.OriRoleIdxTable) == 0 {
			return
		}
		if isUserIn[m.Author.ID] {
			s.ChannelMessageSend(m.ChannelID, "게임에 참가중인 사람은 관전할 수 없습니다.")
			return
		}
		dmChan, _ := s.UserChannelCreate(m.Author.ID)
		g.SendLogMsg(dmChan.ID)
	case config.Prefix + "확인":
		g := guildChanToGameData[m.GuildID+m.ChannelID]
		if g == nil {
			return
		}
		if len(g.OriRoleIdxTable) == 0 {
			return
		}
		Server, _ := s.State.Guild(m.GuildID)
		Channel, _ := s.State.Channel(m.ChannelID)
		msg := "----------------------------------------------------\n"
		msg += "> 현재 서버: " + Server.Name + "\n"
		msg += "> 현재 채널: " + Channel.Name + "\n"
		msg += "> 현재 유저 수: " + strconv.Itoa(len(g.UserList)) + "\n"
		msg += "----------------------------------------------------\n"
		s.ChannelMessageSend(m.ChannelID, msg)
		g.SendLogMsg(m.ChannelID)
	case config.Prefix + "내정보":
		conn, mgctx := util.MongoConn(env)
		user, _ := util.LoadEachUser(m.Author.ID, false, "User", conn.Database("ONUWF"), mgctx)
		if len(user.Nick) == 0 {
			return
		}
		myInfoEmbed := embed.NewEmbed()
		myInfoEmbed.SetTitle("한밤의 늑대인간 유저정보")
		myInfoEmbed.AddField("닉네임", user.Nick)
		if len(user.Title) > 0 {
			myInfoEmbed.AddField("칭호", user.Title)
		}
		myInfoEmbed.AddField("게임횟수", strconv.Itoa(user.CntPlay)+"회")
		myInfoEmbed.AddField("승리횟수", strconv.Itoa(user.CntWin)+"회(승률:"+strconv.Itoa(user.CntWin*100/user.CntPlay)+"%)")
		myInfoEmbed.AddField("최근게임시간", user.RecentGameTime.String())
		myInfoEmbed.AddField("명령어", "ㅁ닉네임")
		dmChan, _ := s.UserChannelCreate(m.Author.ID)
		s.ChannelMessageSendEmbed(dmChan.ID, myInfoEmbed.MessageEmbed)
	case config.Prefix + "닉네임":
		isNickChange[m.Author.ID] = true
		chNick[m.Author.ID] = make(chan string)
		chTimeout := make(chan bool)
		dmChan, _ := s.UserChannelCreate(m.Author.ID)
		s.ChannelMessageSend(dmChan.ID, "닉네임을 변경하려면 5초 안에 입력해주세요.")
		go func(chan bool) {
			time.Sleep(5 * time.Second)
			chTimeout <- true
		}(chTimeout)
		select {
		case nick := <-chNick[m.Author.ID]:
			conn, mgctx := util.MongoConn(env)
			user, _ := util.LoadEachUser(m.Author.ID, false, "User", conn.Database("ONUWF"), mgctx)
			util.SetUserNick(&user, nick, conn.Database("ONUWF"), mgctx)
			s.ChannelMessageSend(dmChan.ID, "닉네임을 "+nick+"으로 변경했습니다.")
			delete(chNick, m.Author.ID)
			isNickChange[m.Author.ID] = false
		case _ = <-chTimeout:
			s.ChannelMessageSend(dmChan.ID, "닉네임을 변경하지 않았습니다.")
			delete(chNick, m.Author.ID)
			isNickChange[m.Author.ID] = false
		}
	}
}

// messageReactionAdd 함수는 인게임 버튼 이모지 상호작용 처리를 위한 이벤트 핸들러 함수입니다.
func messageReactionAdd(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
	// 봇 자기자신의 리액션 무시.
	if r.UserID == s.State.User.ID {
		return
	}
	// 게임 참가중이 아닌 사용자의 리액션 무시.
	// 단, 참가자가 아니면 참가 가능해야 함. 무시해버리면 참가 못 함.
	if !(isUserIn[r.UserID] || (!isUserIn[r.UserID] && r.Emoji.Name == emj["YES"])) {
		s.MessageReactionRemove(r.ChannelID, r.MessageID, r.Emoji.Name, r.UserID)
		return
	}
	g := uidToGameData[r.UserID]
	if g == nil {
		g = guildChanToGameData[r.GuildID+r.ChannelID]
		if g == nil {
			return
		}
	}
	for i := 1; i < 10; i++ {
		emjID := "n" + strconv.Itoa(i)
		if r.Emoji.Name == emj[emjID] {
			go g.CurState.PressNumBtn(s, r.MessageReaction, i)
			break
		}
	}
	switch r.Emoji.Name {
	case emj["DISCARD"]:
		// 🚮
		g.CurState.PressDisBtn(s, r.MessageReaction)
	case emj["YES"]:
		// ⭕️
		g.CurState.PressYesBtn(s, r.MessageReaction)
	case emj["NO"]:
		// ❌
		g.CurState.PressNoBtn(s, r.MessageReaction)
	case emj["LEFT"]:
		// ◀️
		g.CurState.PressDirBtn(s, r.MessageReaction, -1)
	case emj["RIGHT"]:
		// ▶️
		g.CurState.PressDirBtn(s, r.MessageReaction, 1)
	case emj["BOOKMARK"]:
		g.CurState.PressBmkBtn(s, r.MessageReaction)
	}
}
