package handler

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"strings"
	"time"
	"wa-bot/internal/config"
	"wa-bot/internal/utils"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

func (h *BotHandler) HandleCommand(msg *events.Message, text string) {
	parts := strings.Fields(text)
	if len(parts) == 0 { return }

	command := strings.ToLower(parts[0])[1:]
	args := parts[1:]
	start := time.Now()
	sender := msg.Info.Sender.String()
	chat := msg.Info.Chat

	defer func() { utils.LogCommand(sender, command, time.Since(start)) }()

	switch command {
	case "menu", "help":
		menu := `🤖 *BOT DASHBOARD*
👤 *Owner:* Farell Aditya
🔗 *Github:* risqinf

🖼️ *Media Tools*
• /s - Sticker (Smart Fit)
• /toimg - Sticker ke JPG
• /togif - Sticker ke GIF
• /tovid - Sticker ke MP4

🛠️ *System*
• /ping - Cek Speed
• /info - Server Specs
• /afk [alasan] - Set Busy

👮‍♂️ *Group Admin*
• /tagall - Tag semua member
• /kick @user
• /promote @user
• /demote @user
• /setwelcome - Auto welcome
• /role - Admin list`
		h.reply(chat, menu)

	case "ping":
		h.reply(chat, fmt.Sprintf("🏓 Pong!\n⚡ Speed: *%.3fs*", time.Since(start).Seconds()))

	case "info":
		h.reply(chat, "⏳ Mengambil data server...")
		h.reply(chat, getServerInfo(start))

	case "afk":
		reason := "Tanpa alasan"
		if len(args) > 0 { reason = strings.Join(args, " ") }
		config.AFKUsers[msg.Info.Sender.String()] = config.AFKData{Reason: reason, Time: time.Now()}
		h.reply(chat, fmt.Sprintf("💤 *AFK SET*\n\n💬 Alasan: %s", reason))

	case "s", "sticker":
		h.handleStickerCmd(msg, chat)

	case "toimg", "tovid", "togif":
		h.handleConverterCmd(msg, chat, command)

	case "tagall":
		h.handleTagAll(msg, chat)

	case "kick", "promote", "demote":
		h.handleAdminAction(msg, chat, command)
		
	case "setwelcome":
		h.handleSetWelcome(msg, chat, parts, text)
		
	case "role":
		h.handleRole(msg, chat)
	}
}

func (h *BotHandler) handleStickerCmd(msg *events.Message, chat types.JID) {
	var mediaMsg interface{}
	var mediaType string
	
	if msg.Message.GetImageMessage() != nil {
		mediaMsg = msg.Message.GetImageMessage(); mediaType = "image"
	} else if msg.Message.GetVideoMessage() != nil {
		mediaMsg = msg.Message.GetVideoMessage(); mediaType = "video"
	} else {
		quoted := msg.Message.GetExtendedTextMessage().GetContextInfo().GetQuotedMessage()
		if quoted != nil {
			if quoted.GetImageMessage() != nil {
				mediaMsg = quoted.GetImageMessage(); mediaType = "image"
			} else if quoted.GetVideoMessage() != nil {
				mediaMsg = quoted.GetVideoMessage(); mediaType = "video"
			}
		}
	}

	if mediaMsg == nil { h.reply(chat, "❌ Kirim/Reply media"); return }

	utils.RandomDelay()
	
	var data []byte
	var err error
	if mediaType == "image" {
		data, err = h.Client.Download(context.Background(), mediaMsg.(*waProto.ImageMessage))
	} else {
		data, err = h.Client.Download(context.Background(), mediaMsg.(*waProto.VideoMessage))
	}

	if err != nil { h.reply(chat, "❌ Gagal download"); return }

	ext := ".jpg"; if mediaType == "video" { ext = ".mp4" }
	tmpInput := fmt.Sprintf("temp_%d%s", time.Now().UnixNano(), ext)
	ioutil.WriteFile(tmpInput, data, 0644)
	defer os.Remove(tmpInput)

	webpPath, err := convertToWebP(tmpInput, mediaType == "video")
	if err != nil { h.reply(chat, "❌ Gagal convert"); return }
	defer os.Remove(webpPath)

	webpData, _ := ioutil.ReadFile(webpPath)
	uploaded, _ := h.Client.Upload(context.Background(), webpData, whatsmeow.MediaImage)

	h.Client.SendMessage(context.Background(), chat, &waProto.Message{
		StickerMessage: &waProto.StickerMessage{
			URL: &uploaded.URL, DirectPath: &uploaded.DirectPath, MediaKey: uploaded.MediaKey,
			Mimetype: stringPtr("image/webp"), FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256: uploaded.FileSHA256, FileLength: uint64Ptr(uint64(len(webpData))),
		},
	})
}

func (h *BotHandler) handleConverterCmd(msg *events.Message, chat types.JID, cmd string) {
	quoted := msg.Message.GetExtendedTextMessage().GetContextInfo().GetQuotedMessage()
	if quoted == nil || quoted.GetStickerMessage() == nil { h.reply(chat, "❌ Reply sticker!"); return }
	
	stickerMsg := quoted.GetStickerMessage()
	data, err := h.Client.Download(context.Background(), stickerMsg)
	if err != nil { h.reply(chat, "❌ Gagal download"); return }

	tmpWebp := fmt.Sprintf("temp_%d.webp", time.Now().UnixNano())
	ioutil.WriteFile(tmpWebp, data, 0644)
	defer os.Remove(tmpWebp)

	targetFormat := "jpg"
	if cmd == "togif" { targetFormat = "gif" }
	if cmd == "tovid" { targetFormat = "mp4" }

	outPath, err := convertStickerToMedia(tmpWebp, targetFormat)
	if err != nil { h.reply(chat, "❌ Gagal convert"); return }
	defer os.Remove(outPath)

	finalData, _ := ioutil.ReadFile(outPath)
	
	if targetFormat == "jpg" {
		uploaded, _ := h.Client.Upload(context.Background(), finalData, whatsmeow.MediaImage)
		h.Client.SendMessage(context.Background(), chat, &waProto.Message{
			ImageMessage: &waProto.ImageMessage{URL: &uploaded.URL, DirectPath: &uploaded.DirectPath, MediaKey: uploaded.MediaKey, Mimetype: stringPtr("image/jpeg"), FileEncSHA256: uploaded.FileEncSHA256, FileSHA256: uploaded.FileSHA256, FileLength: uint64Ptr(uint64(len(finalData)))},
		})
	} else if targetFormat == "mp4" {
		uploaded, _ := h.Client.Upload(context.Background(), finalData, whatsmeow.MediaVideo)
		h.Client.SendMessage(context.Background(), chat, &waProto.Message{
			VideoMessage: &waProto.VideoMessage{URL: &uploaded.URL, DirectPath: &uploaded.DirectPath, MediaKey: uploaded.MediaKey, Mimetype: stringPtr("video/mp4"), FileEncSHA256: uploaded.FileEncSHA256, FileSHA256: uploaded.FileSHA256, FileLength: uint64Ptr(uint64(len(finalData)))},
		})
	} else {
		// GIF as Document
		uploaded, _ := h.Client.Upload(context.Background(), finalData, whatsmeow.MediaDocument)
		h.Client.SendMessage(context.Background(), chat, &waProto.Message{
			DocumentMessage: &waProto.DocumentMessage{URL: &uploaded.URL, DirectPath: &uploaded.DirectPath, MediaKey: uploaded.MediaKey, Mimetype: stringPtr("image/gif"), FileEncSHA256: uploaded.FileEncSHA256, FileSHA256: uploaded.FileSHA256, FileLength: uint64Ptr(uint64(len(finalData))), FileName: stringPtr("sticker.gif")},
		})
	}
}

func (h *BotHandler) handleTagAll(msg *events.Message, chat types.JID) {
	if !msg.Info.IsGroup || !h.isAdmin(chat, msg.Info.Sender) { return }
	groupInfo, _ := h.Client.GetGroupInfo(context.Background(), chat)
	var mentions []string
	text := "📢 *TAG ALL*\n"
	for _, p := range groupInfo.Participants {
		mentions = append(mentions, p.JID.String())
		text += fmt.Sprintf("@%s\n", p.JID.User)
	}
	h.replyWithMention(chat, text, mentions)
}

func (h *BotHandler) handleAdminAction(msg *events.Message, chat types.JID, cmd string) {
	if !msg.Info.IsGroup || !h.isAdmin(chat, msg.Info.Sender) { return }
	target := getTargetJID(msg)
	if target == "" { return }
	
	jid, _ := types.ParseJID(target)
	if !strings.Contains(target, "@") { jid, _ = types.ParseJID(target + "@s.whatsapp.net") }

	switch cmd {
	case "kick": h.Client.UpdateGroupParticipants(context.Background(), chat, []types.JID{jid}, whatsmeow.ParticipantChangeRemove)
	case "promote": h.Client.UpdateGroupParticipants(context.Background(), chat, []types.JID{jid}, whatsmeow.ParticipantChangePromote)
	case "demote": h.Client.UpdateGroupParticipants(context.Background(), chat, []types.JID{jid}, whatsmeow.ParticipantChangeDemote)
	}
	h.reply(chat, "✅ Done")
}

func (h *BotHandler) handleSetWelcome(msg *events.Message, chat types.JID, parts []string, fullText string) {
	if !msg.Info.IsGroup || !h.isAdmin(chat, msg.Info.Sender) { return }
	groupID := chat.String()
	
	if len(parts) < 2 { return }
	arg := strings.ToLower(parts[1])
	
	c := config.Group.Welcome[groupID]
	if arg == "on" { c.Enabled = true } else if arg == "off" { c.Enabled = false } else {
		startIdx := strings.Index(strings.ToLower(fullText), ".setwelcome") + 11
		if startIdx < len(fullText) {
			c.Message = strings.TrimSpace(fullText[startIdx:])
			c.Enabled = true
		}
	}
	config.Group.Welcome[groupID] = c
	config.SaveGroupConfig()
	h.reply(chat, "✅ Config updated")
}

func (h *BotHandler) handleRole(msg *events.Message, chat types.JID) {
	if !msg.Info.IsGroup { return }
	groupInfo, _ := h.Client.GetGroupInfo(context.Background(), chat)
	res := "👑 *ADMIN LIST*\n"
	for _, p := range groupInfo.Participants {
		if p.IsAdmin || p.IsSuperAdmin { res += fmt.Sprintf("- @%s\n", p.JID.User) }
	}
	h.reply(chat, res)
}

func getServerInfo(start time.Time) string {
	return fmt.Sprintf("💻 *SERVER INFO*\nGO: %s\nCPU: %d Core\nUptime: %s", runtime.Version(), runtime.NumCPU(), utils.FormatDuration(time.Since(start)))
}
