package telegram

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	tg "github.com/amarnathcjd/gogram/telegram"
)

func HandleAddAuth(m *tg.NewMessage) error {
	if !isOwner(m.Sender.ID) {
		m.Reply("<b>Access Denied</b>\n\nOnly the owner can add authorized users.")
		return nil
	}

	args := m.Args()
	if args == "" && !m.IsReply() {
		m.Reply("<b>Usage:</b> <code>/addauth &lt;user_id&gt;</code> or reply to a user with <code>/addauth</code>")
		return nil
	}

	var targetUserID int64
	var username, firstName string

	if m.IsReply() {
		replyMsg, err := m.GetReplyMessage()
		if err == nil {
			targetUserID = replyMsg.SenderID()
			if replyMsg.Sender != nil {
				username = replyMsg.Sender.Username
				firstName = replyMsg.Sender.FirstName
			}
		}
	} else {
		user, err := m.Client.GetSendablePeer(args)
		if err != nil {
			m.Reply("<b>Error:</b> Invalid user. Use <code>/addauth &lt;user_id&gt;</code> or reply to a user.")
			return nil
		}
		targetUserID = m.Client.GetPeerID(user)
	}

	if targetUserID == 0 {
		m.Reply("<b>Error:</b> Could not determine user ID.")
		return nil
	}

	if targetUserID == config.OwnerID {
		m.Reply("<b>Info:</b> Owner is already authorized by default.")
		return nil
	}

	if err := db.AddAuthUser(targetUserID, username, firstName, m.Sender.ID); err != nil {
		log.Printf("[AUTH] Failed to add user %d: %v", targetUserID, err)
		m.Reply("<b>Error:</b> Failed to add user to database.")
		return nil
	}

	authUsersMutex.Lock()
	authUsersCache[targetUserID] = true
	authUsersMutex.Unlock()

	m.Reply(fmt.Sprintf("<b>Success:</b> User <code>%d</code> has been added to authorized users.", targetUserID))
	return nil
}

func HandleRemoveAuth(m *tg.NewMessage) error {
	if !isOwner(m.Sender.ID) {
		m.Reply("<b>Access Denied</b>\n\nOnly the owner can remove authorized users.")
		return nil
	}

	args := strings.Fields(m.Args())
	if len(args) < 1 {
		m.Reply("<b>Usage:</b> <code>/removeauth &lt;user_id&gt;</code> or reply to a user with <code>/removeauth</code>")
		return nil
	}

	var targetUserID int64

	if m.IsReply() {
		replyMsg, err := m.GetReplyMessage()
		if err == nil {
			targetUserID = replyMsg.SenderID()
		}
	} else {
		userID, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			m.Reply("<b>Error:</b> Invalid user ID. Use <code>/removeauth &lt;user_id&gt;</code> or reply to a user.")
			return nil
		}
		targetUserID = userID
	}

	if targetUserID == 0 {
		m.Reply("<b>Error:</b> Could not determine user ID.")
		return nil
	}

	if targetUserID == config.OwnerID {
		m.Reply("<b>Error:</b> Cannot remove owner from authorized users.")
		return nil
	}

	if err := db.RemoveAuthUser(targetUserID); err != nil {
		log.Printf("[AUTH] Failed to remove user %d: %v", targetUserID, err)
		m.Reply("<b>Error:</b> Failed to remove user from database.")
		return nil
	}

	authUsersMutex.Lock()
	delete(authUsersCache, targetUserID)
	authUsersMutex.Unlock()

	m.Reply(fmt.Sprintf("<b>Success:</b> User <code>%d</code> has been removed from authorized users.", targetUserID))
	return nil
}

func HandleListAuth(m *tg.NewMessage) error {
	if !isOwner(m.Sender.ID) {
		m.Reply("<b>Access Denied</b>\n\nOnly the owner can view authorized users.")
		return nil
	}

	authUsers, err := db.GetAllAuthUsers()
	if err != nil {
		log.Printf("[AUTH] Failed to get auth users: %v", err)
		m.Reply("<b>Error:</b> Failed to retrieve authorized users.")
		return nil
	}

	if len(authUsers) == 0 {
		m.Reply("<b>Authorized Users</b>\n\nNo authorized users found.\n\n<b>Owner:</b> <code>" + strconv.FormatInt(config.OwnerID, 10) + "</code>")
		return nil
	}

	var response strings.Builder
	response.WriteString("<b>Authorized Users</b>\n\n")
	response.WriteString(fmt.Sprintf("<b>Owner:</b> <code>%d</code>\n\n", config.OwnerID))

	for i, user := range authUsers {
		response.WriteString(fmt.Sprintf("<b>%d.</b> ", i+1))
		if user.FirstName != "" {
			response.WriteString(user.FirstName)
		}
		if user.Username != "" {
			response.WriteString(fmt.Sprintf(" (@%s)", user.Username))
		}
		response.WriteString(fmt.Sprintf("\n   <b>ID:</b> <code>%d</code>\n", user.UserID))
		response.WriteString(fmt.Sprintf("   <b>Added:</b> %s\n\n", user.CreatedAt.Format("2006-01-02 15:04")))
	}

	m.Reply(response.String())
	return nil
}

func HandleSetPublic(m *tg.NewMessage) error {
	if !isOwner(m.Sender.ID) {
		m.Reply("<b>Access Denied</b>\n\nOnly the owner can change public access settings.")
		return nil
	}

	args := strings.ToLower(strings.TrimSpace(m.Args()))
	if args == "" {
		status := "Private"
		if isPublicAccess() {
			status = "Public"
		}
		m.Reply(fmt.Sprintf("<b>Current Access Mode:</b> %s\n\n<b>Usage:</b>\n<code>/setpublic on</code> - Enable public access\n<code>/setpublic off</code> - Disable public access", status))
		return nil
	}

	var enable bool
	switch args {
	case "on", "true", "enable", "1", "yes":
		enable = true
	case "off", "false", "disable", "0", "no":
		enable = false
	default:
		m.Reply("<b>Error:</b> Invalid option. Use <code>/setpublic on</code> or <code>/setpublic off</code>")
		return nil
	}

	if err := db.SetPublicAccess(enable, m.Sender.ID); err != nil {
		log.Printf("[AUTH] Failed to set public access: %v", err)
		m.Reply("<b>Error:</b> Failed to update public access setting.")
		return nil
	}

	publicAccessMutex.Lock()
	publicAccessCache = enable
	publicAccessMutex.Unlock()

	status := "<b>Private Mode</b>\nOnly owner and authorized users can search."
	if enable {
		status = "<b>Public Mode</b>\nAnyone can use search features."
	}

	m.Reply(fmt.Sprintf("<b>Success:</b> Public access updated.\n\n%s", status))
	return nil
}
