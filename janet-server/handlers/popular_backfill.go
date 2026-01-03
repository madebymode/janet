package handlers

import "time"

type popularBackfillJob struct {
	messageID string
	channelID string
}

func (h *Handler) enqueuePopularMessageBackfill(messageID, channelID string) {
	if messageID == "" || h.slack == nil {
		return
	}

	h.popularBackfillMu.Lock()
	if _, exists := h.popularBackfillSeen[messageID]; exists {
		h.popularBackfillMu.Unlock()
		return
	}
	h.popularBackfillSeen[messageID] = struct{}{}
	h.popularBackfillMu.Unlock()

	select {
	case h.popularBackfillQueue <- popularBackfillJob{messageID: messageID, channelID: channelID}:
	default:
		h.popularBackfillMu.Lock()
		delete(h.popularBackfillSeen, messageID)
		h.popularBackfillMu.Unlock()
		h.notePopularBackfillDrop()
	}
}

func (h *Handler) notePopularBackfillDrop() {
	h.popularDropMu.Lock()
	h.popularDropCount++
	now := time.Now()
	if now.Sub(h.popularDropLastLog) >= 30*time.Second {
		dropped := h.popularDropCount
		h.popularDropCount = 0
		h.popularDropLastLog = now
		h.popularDropMu.Unlock()
		h.logger.KV("dropped", dropped).Info("popular backfill queue full")
		return
	}
	h.popularDropMu.Unlock()
}

func (h *Handler) startPopularMessageBackfillWorker() {
	if h.slack == nil {
		return
	}

	go func() {
		for job := range h.popularBackfillQueue {
			h.processPopularMessageBackfill(job)
		}
	}()
}

func (h *Handler) processPopularMessageBackfill(job popularBackfillJob) {
	defer func() {
		h.popularBackfillMu.Lock()
		delete(h.popularBackfillSeen, job.messageID)
		h.popularBackfillMu.Unlock()
	}()

	if job.messageID == "" || h.slack == nil {
		return
	}

	if cached, err := h.db.GetPopularMessageDetails(job.messageID); err == nil && cached != nil {
		hasText := cached.Text != nil && *cached.Text != ""
		hasPermalink := cached.Permalink != nil && *cached.Permalink != ""
		hasAuthor := (cached.AuthorName != nil && *cached.AuthorName != "") || (cached.AuthorAvatar != nil && *cached.AuthorAvatar != "")
		imageKnown := cached.ImageURL != nil
		if hasText && hasPermalink && hasAuthor && imageKnown {
			return
		}
	}

	channelID := job.channelID
	if channelID == "" {
		if authorUsername, err := h.db.GetMessageAuthorByMessageID(job.messageID); err == nil && authorUsername != nil {
			if found, err := h.slack.FindChannelByMessageAuthorAndTimestamp(*authorUsername, job.messageID); err == nil {
				channelID = found
				if err := h.db.UpdateChannelIDForMessage(job.messageID, found); err != nil {
					h.logger.Err(err).KV("message_id", job.messageID).KV("channel_id", found).Error("failed to cache channel_id to database")
				}
			}
		}
		if channelID == "" {
			if found, err := h.slack.FindChannelByMessageID(job.messageID); err == nil {
				channelID = found
				if err := h.db.UpdateChannelIDForMessage(job.messageID, found); err != nil {
					h.logger.Err(err).KV("message_id", job.messageID).KV("channel_id", found).Error("failed to cache channel_id to database")
				}
			} else {
				return
			}
		}
	}

	details, err := h.slack.GetMessageDetails(channelID, job.messageID)
	if err != nil {
		h.logger.Err(err).KV("message_id", job.messageID).KV("channel_id", channelID).Error("popular backfill failed to get message details")
		return
	}

	var text *string
	var authorID *string
	var authorName *string
	var authorAvatar *string
	var imageURL *string
	var permalink *string
	var isReply *bool
	var isIgnored *bool
	if details != nil {
		if details.Text != "" {
			text = &details.Text
		}
		if details.AuthorID != "" {
			authorID = &details.AuthorID
		}
		if details.AuthorName != "" {
			authorName = &details.AuthorName
		}
		if details.AuthorAvatar != "" {
			authorAvatar = &details.AuthorAvatar
		}
		if details.ImageURL != "" {
			imageURL = &details.ImageURL
		}
		if details.ImageURL == "" && !details.HasImage {
			empty := ""
			imageURL = &empty
		}
		if details.Permalink != "" {
			permalink = &details.Permalink
		}
		if details.IsReply {
			isReplyVal := true
			isReply = &isReplyVal
		} else {
			isReplyVal := false
			isReply = &isReplyVal
		}
		if details.IsIgnored {
			isIgnoredVal := true
			isIgnored = &isIgnoredVal
		} else {
			isIgnoredVal := false
			isIgnored = &isIgnoredVal
		}
	}
	if err := h.db.UpsertPopularMessageDetails(job.messageID, &channelID, text, permalink, authorID, authorName, authorAvatar, imageURL, isReply, isIgnored); err != nil {
		h.logger.Err(err).KV("message_id", job.messageID).Error("failed to cache popular message details")
	}
}
