package handler

import (
	"context"
	"fmt"
	"strings"
	"time"
	"wa-bot/internal/config"
	"wa-bot/internal/utils"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

type BotHandler struct {
	Client    *whatsmeow.Client
	StartTime time.Time
}

func NewBotHandler(c *whatsmeow.Client, startTime time.Time) *BotHandler {
	return &BotHandler{Client: c, StartTime: startTime}
}

func (h *BotHandler) EventHandler(evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		h.HandleMessage(v)
	case *events.GroupInfo:
		h.HandleGroupEvent(v)
	}
}

func (h *BotHandler) HandleMessage(msg *events.Message) {
	var text string
	msgType := "Text"

	if msg.Message.GetConversation() != "" {
		text = msg.Message.GetConversation()
	} else if msg.Message.GetExtendedTextMessage() != nil {
		text = msg.Message.GetExtendedTextMessage().GetText()
	} else if msg.Message.GetImageMessage() != nil {
		text = msg.Message.GetImageMessage().GetCaption(); msgType = "Image"
	} else if msg.Message.GetVideoMessage() != nil {
		text = msg.Message.GetVideoMessage().GetCaption(); msgType = "Video"
	} else if msg.Message.GetStickerMessage() != nil {
		msgType = "Sticker"; text = "[Sticker]"
	}

	senderJID := msg.Info.Sender.String()
	chatJID := msg.Info.Chat.String()
	senderName := strings.Split(senderJID, "@")[0]
	targetName := strings.Split(chatJID, "@")[0]

	if msg.Info.IsGroup { targetName = "Group-" + targetName }

	utils.LogRealtime(senderName, targetName, msgType, text, msg.Info.IsFromMe)

	if afkInfo, exists := config.AFKUsers[senderJID]; exists {
		duration := time.Since(afkInfo.Time)
		delete(config.AFKUsers, senderJID)
		go h.reply(msg.Info.Chat, fmt.Sprintf("👋 Welcome back!\nAFK selama: %s", utils.FormatDuration(duration)))
	}

	if msg.Info.IsGroup { go h.checkAFKMentions(msg) }

	if len(text) > 1 && (strings.HasPrefix(text, "/") || strings.HasPrefix(text, ".")) {
		go h.HandleCommand(msg, text)
	}
}

func (h *BotHandler) HandleGroupEvent(evt *events.GroupInfo) {
	if len(evt.Join) > 0 {
		c, exists := config.Group.Welcome[evt.JID.String()]
		if !exists || !c.Enabled { return }
		time.Sleep(1 * time.Second)
		for _, p := range evt.Join {
			msg := strings.ReplaceAll(c.Message, "{user}", "@"+p.User)
			h.replyWithMention(evt.JID, msg, []string{p.String()})
		}
	}
}

func (h *BotHandler) reply(jid types.JID, text string) {
	h.Client.SendChatPresence(context.Background(), jid, types.ChatPresenceComposing, types.ChatPresenceMediaText)
	utils.RandomDelay()
	h.Client.SendMessage(context.Background(), jid, &waProto.Message{Conversation: &text})
}

func (h *BotHandler) replyWithMention(jid types.JID, text string, mentions []string) {
	h.Client.SendMessage(context.Background(), jid, &waProto.Message{
		ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text: &text, ContextInfo: &waProto.ContextInfo{MentionedJID: mentions},
		},
	})
}

func (h *BotHandler) checkAFKMentions(msg *events.Message) {
	extMsg := msg.Message.GetExtendedTextMessage()
	if extMsg == nil { return }
	for _, mentioned := range extMsg.GetContextInfo().GetMentionedJID() {
		if afkData, exists := config.AFKUsers[mentioned]; exists {
			h.replyWithMention(msg.Info.Chat, fmt.Sprintf("💤 @%s sedang AFK\nReason: %s", strings.Split(mentioned, "@")[0], afkData.Reason), []string{mentioned})
			break
		}
	}
}

func (h *BotHandler) isAdmin(group, user types.JID) bool {
	g, err := h.Client.GetGroupInfo(context.Background(), group)
	if err != nil { return false }
	for _, p := range g.Participants {
		if p.JID.User == user.User { return p.IsAdmin || p.IsSuperAdmin }
	}
	return false
}

func getTargetJID(msg *events.Message) string {
	if msg.Message.GetExtendedTextMessage() != nil {
		m := msg.Message.GetExtendedTextMessage().GetContextInfo().GetMentionedJID()
		if len(m) > 0 { return m[0] }
		q := msg.Message.GetExtendedTextMessage().GetContextInfo().GetQuotedMessage()
		if q != nil { return msg.Message.GetExtendedTextMessage().GetContextInfo().GetParticipant() }
	}
	return ""
}

func stringPtr(s string) *string { return &s }
func uint64Ptr(i uint64) *uint64 { return &i }
