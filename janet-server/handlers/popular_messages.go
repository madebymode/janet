package handlers

import (
	"net/http"
	"sort"
	"strings"

	"github.com/troyxmccall/janet/database"
)

type popularMessagesQuery struct {
	Limit        int
	Offset       int
	Year         int
	FilterUser   string
	MinReactions int
	MediaOnly    bool
	FunnyBias    bool
	IncludeMeta  bool
}

type popularMessageEntry struct {
	data          map[string]interface{}
	reactionCount int
}

type popularMessagesResult struct {
	Items               []map[string]interface{}
	Total               int
	Pending             int
	QueuePosition       int
	SkippedReplies      int
	SkippedIgnored      int
	SkippedTestChannels int
	BackfillEnqueued    int
}

func parsePopularMessagesQuery(r *http.Request) (*popularMessagesQuery, error) {
	year, err := parseOptionalYear(r)
	if err != nil {
		return nil, err
	}

	filterUser, err := sanitizeUsernameFilter(r.URL.Query().Get("user"))
	if err != nil {
		return nil, err
	}

	return &popularMessagesQuery{
		Limit:        parseIntQuery(r, "limit", 15, 1, 15),
		Offset:       parseIntQuery(r, "offset", 0, 0, 100000),
		Year:         year,
		FilterUser:   filterUser,
		MinReactions: parseIntQuery(r, "min_reactions", 0, 0, 100000),
		MediaOnly:    parseBoolQuery(r, "has_media"),
		FunnyBias:    parseBoolQuery(r, "funny_bias"),
		IncludeMeta:  parseBoolQuery(r, "include_meta"),
	}, nil
}

func (h *Handler) buildPopularMessagesResult(query popularMessagesQuery) (*popularMessagesResult, error) {
	messages, err := h.fetchPopularMessages(query)
	if err != nil {
		return nil, err
	}

	result := &popularMessagesResult{}
	entries := make([]popularMessageEntry, 0, len(messages))

	for _, msg := range messages {
		entry, skipReason, err := h.buildPopularMessageEntry(msg, query)
		if err != nil {
			return nil, err
		}

		switch skipReason {
		case "missing":
			result.Pending++
			result.BackfillEnqueued++
		case "reply":
			result.SkippedReplies++
			continue
		case "ignored":
			result.SkippedIgnored++
			continue
		case "test_channel":
			result.SkippedTestChannels++
			continue
		}

		if entry == nil {
			continue
		}
		entries = append(entries, *entry)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].reactionCount > entries[j].reactionCount
	})

	result.Total = h.getPopularMessagesTotal(query, len(entries))
	start, end := paginate(query.Offset, query.Limit, len(entries))
	result.Items = make([]map[string]interface{}, 0, end-start)
	for _, entry := range entries[start:end] {
		result.Items = append(result.Items, entry.data)
	}
	result.QueuePosition = h.minPopularQueuePosition(result.Items)

	return result, nil
}

func (h *Handler) fetchPopularMessages(query popularMessagesQuery) ([]*database.PopularMessage, error) {
	fetchLimit := (query.Offset + query.Limit) * 10
	minFetch := 500
	if query.FilterUser != "" {
		minFetch = 2000
	}
	if fetchLimit < minFetch {
		fetchLimit = minFetch
	}
	if fetchLimit > 5000 {
		fetchLimit = 5000
	}

	if query.FilterUser != "" {
		return h.db.GetPopularMessagesByUser(fetchLimit, query.Year, query.FilterUser, query.FunnyBias)
	}
	return h.db.GetPopularMessages(fetchLimit, query.Year, query.FunnyBias)
}

func (h *Handler) buildPopularMessageEntry(msg *database.PopularMessage, query popularMessagesQuery) (*popularMessageEntry, string, error) {
	msgData := map[string]interface{}{
		"channel_id":     msg.ChannelID,
		"message_id":     msg.MessageID,
		"reaction_count": msg.ReactionCount,
	}

	channelID := ""
	if msg.ChannelID != nil && *msg.ChannelID != "" {
		channelID = *msg.ChannelID
	}

	hasText := false
	hasPermalink := false
	hasAuthor := false
	imageKnown := false
	attachmentKnown := false
	reactionKnown := false
	detailsFetched := false
	detailsFetchedKnown := false

	if cached, err := h.db.GetPopularMessageDetails(msg.MessageID); err == nil && cached != nil {
		if cached.ChannelID != nil && channelID == "" {
			channelID = *cached.ChannelID
			msgData["channel_id"] = cached.ChannelID
		}
		if cached.Text != nil && *cached.Text != "" {
			msgData["text"] = *cached.Text
			hasText = true
		}
		if cached.Permalink != nil && *cached.Permalink != "" {
			msgData["permalink"] = *cached.Permalink
			hasPermalink = true
		}
		if cached.AuthorName != nil && *cached.AuthorName != "" {
			msgData["author_name"] = *cached.AuthorName
			hasAuthor = true
		}
		if cached.AuthorAvatar != nil && *cached.AuthorAvatar != "" {
			msgData["author_avatar"] = *cached.AuthorAvatar
			hasAuthor = true
		}
		if cached.ImageURL != nil {
			imageKnown = true
			if *cached.ImageURL != "" {
				msgData["image_url"] = *cached.ImageURL
			}
		}
		if cached.AttachmentURL != nil && *cached.AttachmentURL != "" {
			msgData["attachment_url"] = *cached.AttachmentURL
		}
		if cached.AttachmentURL != nil {
			attachmentKnown = true
			if cached.AttachmentMime != nil && *cached.AttachmentMime != "" {
				msgData["attachment_mime"] = *cached.AttachmentMime
			}
		}
		if cached.ImageURL == nil && cached.AttachmentURL == nil && cached.AttachmentMime == nil {
			imageKnown = true
			attachmentKnown = true
		}
		if cached.ReactionCount != nil {
			reactionKnown = true
			cachedCount := *cached.ReactionCount
			if cachedCount < msg.ReactionCount {
				cachedCount = msg.ReactionCount
			}
			msgData["reaction_count"] = cachedCount
		}
		if cached.IsReply != nil && *cached.IsReply {
			return nil, "reply", nil
		}
		if cached.IsIgnored != nil && *cached.IsIgnored {
			return nil, "ignored", nil
		}
		if cached.DetailsFetched != nil {
			detailsFetchedKnown = true
			detailsFetched = *cached.DetailsFetched
		}
	} else if err != nil {
		h.logger.Err(err).KV("message_id", msg.MessageID).Error("failed to read popular message cache")
	}

	if channelID == "" {
		if storedChannelID, err := h.db.GetChannelIDForMessage(msg.MessageID); err == nil && storedChannelID != nil {
			channelID = *storedChannelID
			msgData["channel_id"] = storedChannelID
		} else if err != nil {
			h.logger.Err(err).KV("message_id", msg.MessageID).Error("failed to resolve channel_id from transactions")
		}
	}

	if strings.HasPrefix(channelID, "TEST") {
		return nil, "test_channel", nil
	}

	if !hasAuthor {
		if derivedAuthor, err := h.db.GetMessageAuthorByMessageID(msg.MessageID); err == nil && derivedAuthor != nil {
			msgData["author_name"] = *derivedAuthor
			hasAuthor = true
			if err := h.db.UpsertPopularMessageDetails(msg.MessageID, nil, nil, nil, nil, derivedAuthor, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
				h.logger.Err(err).KV("message_id", msg.MessageID).Error("failed to cache derived author")
			}
		} else if err != nil {
			h.logger.Err(err).KV("message_id", msg.MessageID).Error("failed to derive author from transactions")
		}
	}

	if channelID != "" {
		cachedChannelID := channelID
		if err := h.db.UpsertPopularMessageDetails(msg.MessageID, &cachedChannelID, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
			h.logger.Err(err).KV("message_id", msg.MessageID).Error("failed to update popular message cache")
		}
	}

	completeDetails := hasText && hasPermalink && hasAuthor && imageKnown && attachmentKnown && reactionKnown
	if !detailsFetchedKnown && completeDetails {
		detailsFetched = true
		detailsFetchedKnown = true
		if err := h.db.UpsertPopularMessageDetails(msg.MessageID, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, &detailsFetched); err != nil {
			h.logger.Err(err).KV("message_id", msg.MessageID).Error("failed to mark popular message details fetched")
		}
	}

	missingDetails := !detailsFetchedKnown || !detailsFetched
	if missingDetails {
		h.enqueuePopularMessageBackfill(msg.MessageID, channelID)
		if position := h.popularBackfillQueuePosition(msg.MessageID); position > 0 {
			msgData["queue_position"] = position
		}
	}
	msgData["pending_details"] = missingDetails

	reactionCount := msg.ReactionCount
	if cachedCount, ok := msgData["reaction_count"].(int); ok {
		reactionCount = cachedCount
	}

	if query.FilterUser != "" {
		author, _ := msgData["author_name"].(string)
		if author != query.FilterUser {
			return nil, "", nil
		}
	}
	if query.MinReactions > 0 && reactionCount < query.MinReactions {
		return nil, "", nil
	}
	if query.MediaOnly {
		_, hasImage := msgData["image_url"]
		_, hasAttachment := msgData["attachment_url"]
		if !hasImage && !hasAttachment {
			return nil, "", nil
		}
	}

	if missingDetails {
		return &popularMessageEntry{data: msgData, reactionCount: reactionCount}, "missing", nil
	}

	return &popularMessageEntry{data: msgData, reactionCount: reactionCount}, "", nil
}

func (h *Handler) getPopularMessagesTotal(query popularMessagesQuery, fallback int) int {
	if query.MediaOnly {
		if total, err := h.db.GetPopularMessageCountWithMedia(query.Year, query.FilterUser, query.MinReactions, query.FunnyBias); err == nil {
			return total
		} else {
			h.logger.Err(err).Error("failed to get popular message count")
		}
		return fallback
	}

	if total, err := h.db.GetPopularMessageCount(query.Year, query.FilterUser, query.MinReactions, query.FunnyBias); err == nil {
		return total
	} else {
		h.logger.Err(err).Error("failed to get popular message count")
	}
	return fallback
}

func (h *Handler) minPopularQueuePosition(items []map[string]interface{}) int {
	queuePosition := 0
	for _, item := range items {
		pending, _ := item["pending_details"].(bool)
		if !pending {
			continue
		}
		position, _ := item["queue_position"].(int)
		if position == 0 {
			messageID, _ := item["message_id"].(string)
			if messageID == "" {
				continue
			}
			position = h.popularBackfillQueuePosition(messageID)
		}
		if position == 0 {
			continue
		}
		if queuePosition == 0 || position < queuePosition {
			queuePosition = position
		}
	}
	return queuePosition
}

func paginate(offset, limit, total int) (int, int) {
	start := offset
	if start > total {
		start = total
	}
	end := start + limit
	if end > total {
		end = total
	}
	return start, end
}
