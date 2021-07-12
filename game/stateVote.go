package game

import (
	"fmt"
	"strconv"

	"github.com/bwmarrin/discordgo"
	embed "github.com/clinet/discordgo-embed"
)

// Prepare is test
type StateVote struct {
	G          *Game
	Voted_list []int
	User_num   int
	Vote_count int
}

// PressNumBtn 사용자가 숫자 이모티콘을 눌렀을 때 state에서 하는 동작
func (v *StateVote) PressNumBtn(s *discordgo.Session, r *discordgo.MessageReactionAdd, num int) {
	//num를 받음
	//해당 index list count +1
	v.Voted_list[num-1]++
	fmt.Println(v.Voted_list[num-1])
	s.ChannelMessageDelete(r.ChannelID, r.MessageID)
	v.Vote_count++
	if v.Vote_count == v.User_num {
		max_value := 0
		for i := 0; i < v.User_num; i++ {
			if max_value < v.Voted_list[i] {
				max_value = v.Voted_list[i]
			}
		}
		voteResultEmbed := embed.NewEmbed()
		voteResultEmbed.SetTitle("투표 결과")
		for i := 0; i < v.User_num; i++ {
			if max_value == v.Voted_list[i] {
				voteResultEmbed.AddField(v.G.UserList[i].nick, v.G.UserList[i].nick+"는 투표로 사망하였습니다.")
			}
		}
		s.ChannelMessageSendEmbed(v.G.ChanID, voteResultEmbed.MessageEmbed)
	}
}

// PressDisBtn 사용자가 버려진 카드 이모티콘을 눌렀을 때 state에서 하는 동작
func (v *StateVote) PressDisBtn(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
	//do nothing
}

// PressYesBtn 사용자가 yes 이모티콘을 눌렀을 때 state에서 하는 동작
func (v *StateVote) PressYesBtn(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
	//do nothing
}

// PressNoBtn 사용자가 No 이모티콘을 눌렀을 때 state에서 하는 동작
func (v *StateVote) PressNoBtn(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
	//do nothing
}

// PressDirBtn 좌 -1, 우 1 사용자가 좌우 방향 이모티콘을 눌렀을 때 state에서 하는 동작
func (v *StateVote) PressDirBtn(s *discordgo.Session, r *discordgo.MessageReactionAdd, dir int) {
	fmt.Println(dir, "test")
	//do nothing
}

// InitState 함수는 스테이트가 시작할 때 필요한 메세지를 생성하고 채널이나 개인DM으로 메세지를 보낸 후
// 메세지 객체를 스테이트의 멤버로 저장합니다.
// 이 함수는 이전 스테이트가 끝나는 시점에 호출되어야 합니다.
func (v *StateVote) InitState() {

}

// stateFinish 함수는 현재 state가 끝나고 다음 state로 넘어갈 때 호출되는 함수입니다.
// game의 CurState 변수에 다음 state를 생성해서 할당해준 다음
// 다음 state의 InitState() 함수를 이 함수 안에서 호출해야 합니다
func (v *StateVote) stateFinish() {

}

// filterReaction 함수는 각 스테이트에서 보낸 메세지에 리액션 했는지 거르는 함수이다.
// 각 스테이트에서 보낸 메세지의 아이디와 리액션이 온 아이디가 동일한지 확인 및
// 메세지에 리액션 한 것을 지워주어야 한다.
func (v *StateVote) filterReaction(s *discordgo.Session, r *discordgo.MessageReactionAdd) {

}

func VoteProcess(s *discordgo.Session, g *Game) {
	//send msg
	//개별로 각 채널에서 수행하게 해야함
	//user vote event handler
	//참가자마다 입력을 처리 *go routine 입력을 state vote preesnumbtn 이용해서 count
	// 결과 값을 visualization (통합채널)
	num := len(g.UserList)
	for i := 0; i < num; i++ {
		go SendVoteDM(s, g, i)
	}

}

func SendVoteDM(s *discordgo.Session, g *Game, UserNum int) {
	voteEmbed := embed.NewEmbed()
	voteEmbed.SetTitle("투표")
	voteEmbed.SetDescription("늑대인간으로 의심되는 대상에게 투표해주세요")
	num := len(g.UserList)
	for i := 0; i < num-1; i++ {
		//이후에 본인 빼도록 수정해야함
		j := i
		if j >= UserNum {
			j = j + 1
		}
		voteEmbed.AddField(strconv.Itoa(i+1)+"번 ", g.UserList[j].nick)
	}
	voteEmbed.SetAuthor(g.UserList[UserNum].nick)
	UserDM, _ := s.UserChannelCreate(g.UserList[0].UserID) //g.UserList[0] -> g.UsrList[UserNum] change need(test용)
	voteMsg, _ := s.ChannelMessageSendEmbed(UserDM.ID, voteEmbed.MessageEmbed)
	addNumAddEmoji(s, voteMsg, g)
}

func addNumAddEmoji(s *discordgo.Session, msg *discordgo.Message, g *Game) {
	num := len(g.UserList)
	for i := 0; i < num-1; i++ {
		s.MessageReactionAdd(msg.ChannelID, msg.ID, g.Emj["n"+strconv.Itoa(i+1)])
	}
	//s.MessageReactionAdd(msg.ChannelID, msg.ID, g.Emj["n2"])
	//s.MessageReactionAdd(msg.ChannelID, msg.ID, g.Emj["n3"])
	//s.MessageReactionAdd(msg.ChannelID, msg.ID, g.Emj["n4"])
}