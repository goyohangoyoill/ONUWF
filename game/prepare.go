package game

import (
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	embed "github.com/clinet/discordgo-embed"
)

type Prepare struct {
	// state에서 가지고 있는 game
	g *Game

	// factory 에서 쓰이게 될 role index
	roleIndex int

	// 직업추가 확인용 메세지
	RoleAddMsg *discordgo.Message

	// 게임입장 확인용 메세지
	EnterGameMsg *discordgo.Message
}

// PressNumBtn 사용자가 숫자 이모티콘을 눌렀을 때 Prepare에서 하는 동작
func (sPrepare *Prepare) PressNumBtn(s *discordgo.Session, r *discordgo.MessageReaction, num int) {
	// do nothing
}

// PressDisBtn 사용자가 버려진 카드 이모티콘을 눌렀을 때 Prepare에서 하는 동작
func (sPrepare *Prepare) PressDisBtn(s *discordgo.Session, r *discordgo.MessageReaction) {
	// do nothing
}

// PressYesBtn 사용자가 yes 이모티콘을 눌렀을 때 Prepare에서 하는 동작
func (sPrepare *Prepare) PressYesBtn(s *discordgo.Session, r *discordgo.MessageReaction) {
	// 게임 진행과 관련된 메세지에 달린 리액션 지운다
	if sPrepare.filterReaction(s, r) {
		return
	}
	// 입장 메세지에서 리액션한거라면
	if r.MessageID == sPrepare.EnterGameMsg.ID {
		//user 생성해서 append()
		sPrepare.g.SetUserByID(r.UserID)
		// 입장 확인 메세지 반영
		s.ChannelMessageEditEmbed(sPrepare.g.ChanID, sPrepare.EnterGameMsg.ID, sPrepare.NewEnterEmbed().MessageEmbed)
		// 직업추가 메세지에서 리액션한거라면
	} else if r.MessageID == sPrepare.RoleAddMsg.ID && r.UserID == sPrepare.g.MasterID {
		// role 생성해서 game의 RoleView와 RoleSeq에 추가
		sPrepare.g.AddRole(sPrepare.roleIndex)
	}
	// 직업 추가 메세지 반영
	s.ChannelMessageEditEmbed(sPrepare.g.ChanID, sPrepare.RoleAddMsg.ID, sPrepare.NewRoleEmbed().MessageEmbed)
	s.MessageReactionRemove(sPrepare.g.ChanID, r.MessageID, r.Emoji.Name, r.UserID)
}

// PressNoBtn 사용자가 No 이모티콘을 눌렀을 때 Prepare에서 하는 동작
func (sPrepare *Prepare) PressNoBtn(s *discordgo.Session, r *discordgo.MessageReaction) {
	// 게임 진행과 관련된 메세지에 달린 리액션 지운다
	if sPrepare.filterReaction(s, r) {
		return
	}
	// 입장 메세지에서 리액션한거라면
	if r.MessageID == sPrepare.EnterGameMsg.ID {
		// userList에서 지우고
		sPrepare.g.DelUserByID(r.UserID)
		// 직업추가 메세지에서 리액션한거라면
	} else if r.MessageID == sPrepare.RoleAddMsg.ID && r.UserID == sPrepare.g.MasterID {
		if sPrepare.roleIndex == 2 {
			s.ChannelMessageSend(sPrepare.g.ChanID, "늑대인간은 2개 있어야 합니다")
			return
		}
		// role 생성해서 game의 RoleView와 RoleSeq에서 찾아 제거
		sPrepare.g.DelRole(sPrepare.roleIndex)
	}
	// 입장 확인 메세지 반영
	s.ChannelMessageEditEmbed(sPrepare.g.ChanID, sPrepare.EnterGameMsg.ID, sPrepare.NewEnterEmbed().MessageEmbed)
	// 직업 추가 메세지 반영
	s.ChannelMessageEditEmbed(sPrepare.g.ChanID, sPrepare.RoleAddMsg.ID, sPrepare.NewRoleEmbed().MessageEmbed)
	s.MessageReactionRemove(sPrepare.g.ChanID, r.MessageID, r.Emoji.Name, r.UserID)
}

// PressDirBtn 좌 -1, 우 1 사용자가 좌우 방향 이모티콘을 눌렀을 때 Prepare에서 하는 동작
func (sPrepare *Prepare) PressDirBtn(s *discordgo.Session, r *discordgo.MessageReaction, dir int) {
	// 게임 진행과 관련된 메세지에 달린 리액션 지운다
	if sPrepare.filterReaction(s, r) {
		return
	}
	// 입장 메세지에서 리액션한거라면
	if r.MessageID == sPrepare.EnterGameMsg.ID {
		// 게임 시작
		if dir == 1 && len(sPrepare.g.RoleView) == len(sPrepare.g.UserList)+3 {
			sPrepare.stateFinish()
		}
		// 직업추가 메세지에서 리액션한거라면
	} else if r.MessageID == sPrepare.RoleAddMsg.ID && r.UserID == sPrepare.g.MasterID {
		// roleindex 증감
		sPrepare.roleIndex += dir
		if sPrepare.roleIndex >= len(sPrepare.g.RG) {
			sPrepare.roleIndex = 0
		} else if sPrepare.roleIndex < 0 {
			sPrepare.roleIndex = len(sPrepare.g.RG) - 1
		}
		// 직업 추가 메세지 반영
		s.ChannelMessageEditEmbed(sPrepare.g.ChanID, sPrepare.RoleAddMsg.ID, sPrepare.NewRoleEmbed().MessageEmbed)
	}
	s.MessageReactionRemove(sPrepare.g.ChanID, r.MessageID, r.Emoji.Name, r.UserID)
}

// InitState 함수는 prepare state가 시작할 때 입장, 직업추가 메세지를 보냅니다.
func (sPrepare *Prepare) InitState() {
	// 늑대인간 2개 추가
	sPrepare.g.AddRole(2)
	sPrepare.g.AddRole(2)

	// <test code>
	//sPrepare.g.AddRole(0)
	//sPrepare.g.AddRole(1)
	//sPrepare.g.AddRole(7)
	//sPrepare.g.AddRole(8)
	//sPrepare.g.AddRole(9)

	enterEmbed := sPrepare.NewEnterEmbed()
	roleEmbed := sPrepare.NewRoleEmbed()
	s := sPrepare.g.Session
	sPrepare.EnterGameMsg, _ = s.ChannelMessageSendEmbed(sPrepare.g.ChanID, enterEmbed.MessageEmbed)
	// 게임 입장 메시지에 안내 버튼을 연결
	s.MessageReactionAdd(sPrepare.EnterGameMsg.ChannelID, sPrepare.EnterGameMsg.ID, sPrepare.g.Emj["YES"])
	s.MessageReactionAdd(sPrepare.EnterGameMsg.ChannelID, sPrepare.EnterGameMsg.ID, sPrepare.g.Emj["NO"])
	s.MessageReactionAdd(sPrepare.EnterGameMsg.ChannelID, sPrepare.EnterGameMsg.ID, sPrepare.g.Emj["RIGHT"])
	sPrepare.RoleAddMsg, _ = s.ChannelMessageSendEmbed(sPrepare.g.ChanID, roleEmbed.MessageEmbed)
	// 직업 추가 메시지에 안내 버튼을 연결
	s.MessageReactionAdd(sPrepare.RoleAddMsg.ChannelID, sPrepare.RoleAddMsg.ID, sPrepare.g.Emj["YES"])
	s.MessageReactionAdd(sPrepare.RoleAddMsg.ChannelID, sPrepare.RoleAddMsg.ID, sPrepare.g.Emj["NO"])
	s.MessageReactionAdd(sPrepare.RoleAddMsg.ChannelID, sPrepare.RoleAddMsg.ID, sPrepare.g.Emj["LEFT"])
	s.MessageReactionAdd(sPrepare.RoleAddMsg.ChannelID, sPrepare.RoleAddMsg.ID, sPrepare.g.Emj["RIGHT"])
}

func (sPrepare *Prepare) stateFinish() {
	sPrepare.g.CurState = &StartGame{sPrepare.g}
	msg, _ := sPrepare.g.Session.ChannelMessageSend(sPrepare.g.ChanID, "각자의 직업을 배정 중입니다...")
	sPrepare.g.GameStateMID = msg.ID
	sPrepare.g.GameStartedChan <- true
	sPrepare.g.CurState.InitState()
}

// filterReaction 함수는 입장 메세지랑 직업추가 메세지에 리액션한게 아니면 걸러준다.
func (sPrepare *Prepare) filterReaction(s *discordgo.Session, r *discordgo.MessageReaction) bool {
	// 현재 스테이트에서 보낸 메세지에 리액션한 게 아니면 거름
	if !(r.MessageID == sPrepare.EnterGameMsg.ID || r.MessageID == sPrepare.RoleAddMsg.ID) {
		return true
	}
	if nil == sPrepare.g.FindUserByUID(r.UserID) && !(r.MessageID == sPrepare.EnterGameMsg.ID && r.Emoji.Name == sPrepare.g.Emj["YES"]) {
		return true
	}
	return false
}

// newRoleEmbed 함수는 role guide와 현재 게임에 추가된 직업 / 게임의 참여중인 인원수 + 3 임베드를 만든다
func (sPrepare *Prepare) NewRoleEmbed() *embed.Embed {
	roleEmbed := embed.NewEmbed()
	title := "직업 추가" + "（" + strconv.Itoa(sPrepare.roleIndex+1) + "／" + strconv.Itoa(len(sPrepare.g.RG)) + "）"
	roleEmbed.SetTitle(title)
	roleEmbed.AddField(sPrepare.g.RG[sPrepare.roleIndex].RoleName, strings.Join(sPrepare.g.RG[sPrepare.roleIndex].RoleGuide, "\n"))
	roleStr := ""
	if len(sPrepare.g.RoleView) == 0 {
		roleStr += "*추가된 직업이 없습니다.*"
	} else {
		for _, item := range sPrepare.g.RoleSeq {
			cnt := sPrepare.g.RoleCount(item, sPrepare.g.RoleView)
			roleStr += item.String() + " " + strconv.Itoa(cnt) + "개"
			if cnt == sPrepare.g.RG[item.ID()].Max {
				roleStr += " 최대"
			}
			roleStr += "\n"
		}
	}
	roleEmbed.AddField("추가된 직업", roleStr)
	roleEmbed.SetFooter("현재 인원에 맞는 직업 수: " + strconv.Itoa(len(sPrepare.g.RoleView)) + " / " + strconv.Itoa(len(sPrepare.g.UserList)+3))
	return roleEmbed
}

// newEnterEmbed 함수는 게임 참여자 목록 임베드를 만든다
func (sPrepare *Prepare) NewEnterEmbed() *embed.Embed {
	enterEmbed := embed.NewEmbed()
	enterEmbed.SetTitle("게임 참가")
	enterStr := ""
	for _, item := range sPrepare.g.UserList {
		enterStr += "`" + item.nick + "`\n"
	}
	enterEmbed.AddField("참가자 목록", "현재 참가 인원: "+strconv.Itoa(len(sPrepare.g.UserList))+"명\n"+enterStr)
	enterEmbed.SetFooter("(최대 10명, 방장은 나갈 수 없음)\n⭕: 입장 ❌: 퇴장 ▶️: 시작")
	return enterEmbed
}
