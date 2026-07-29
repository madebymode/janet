package handlers

import "time"

type popularBackfillJob struct {
	messageID    string
	channelID    string
	forceRefresh bool
}

func (h *Handler) enqueuePopularMessageBackfill(messageID, channelID string) {
	h.enqueuePopularMessageJob(messageID, channelID, false, false)
}

func (h *Handler) enqueuePopularMessageRefresh(messageID, channelID string) {
	h.enqueuePopularMessageJob(messageID, channelID, true, true)
}

func (h *Handler) enqueuePopularMessageJob(messageID, channelID string, forceRefresh, waitForSpace bool) {
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

	h.popularBackfillOrderMu.Lock()
	for {
		select {
		case h.popularBackfillQueue <- popularBackfillJob{messageID: messageID, channelID: channelID, forceRefresh: forceRefresh}:
			h.popularBackfillOrder++
			h.popularBackfillPositions[messageID] = h.popularBackfillOrder
			h.popularBackfillOrderMu.Unlock()
			h.notePopularQueueStatus()
			return
		default:
			if !waitForSpace {
				h.popularBackfillOrderMu.Unlock()
				h.popularBackfillMu.Lock()
				delete(h.popularBackfillSeen, messageID)
				h.popularBackfillMu.Unlock()
				h.notePopularBackfillDrop()
				return
			}
		}
		h.popularBackfillOrderMu.Unlock()
		time.Sleep(1 * time.Second)
		h.popularBackfillOrderMu.Lock()
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
		h.logger.KV("dropped", dropped).Error("popular backfill queue full")
		return
	}
	h.popularDropMu.Unlock()
}

func (h *Handler) notePopularBackfillFailure(reason, messageID string, err error) {
	h.popularFailMu.Lock()
	h.popularFailCounts[reason]++
	count := h.popularFailCounts[reason]
	now := time.Now()
	shouldLog := count == 1 || now.Sub(h.popularFailLastLog) >= 30*time.Second
	if shouldLog {
		h.popularFailLastLog = now
	}
	h.popularFailMu.Unlock()

	if shouldLog {
		logEvent := h.logger.KV("reason", reason).KV("message_id", messageID).KV("count", count)
		if err != nil {
			logEvent = logEvent.Err(err)
		}
		logEvent.Error("popular backfill failed")
	}
}

func (h *Handler) notePopularQueueStatus() {
	h.popularQueueLogMu.Lock()
	now := time.Now()
	if now.Sub(h.popularQueueLastLog) < 30*time.Second {
		h.popularQueueLogMu.Unlock()
		return
	}
	h.popularQueueLastLog = now
	h.popularQueueLogMu.Unlock()
	h.logger.KV("queue_size", h.popularBackfillQueueSize()).KV("queue_processed", h.popularBackfillProcessed).Info("popular_queue status")
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
	queuePosition := 0
	h.popularBackfillOrderMu.Lock()
	if _, ok := h.popularBackfillPositions[job.messageID]; ok {
		order := h.popularBackfillPositions[job.messageID]
		position := int(order - h.popularBackfillProcessed)
		if position > 0 {
			queuePosition = position
		}
		delete(h.popularBackfillPositions, job.messageID)
	}
	h.popularBackfillProcessed++
	h.popularBackfillOrderMu.Unlock()
	h.notePopularQueueStatus()
	h.logger.KV("message_id", job.messageID).KV("queue_size", h.popularBackfillQueueSize()).KV("queue_position", queuePosition).Info("popular_queue job_start")

	defer func() {
		h.popularBackfillMu.Lock()
		delete(h.popularBackfillSeen, job.messageID)
		h.popularBackfillMu.Unlock()
	}()

	if job.messageID == "" || h.slack == nil {
		return
	}

	if !job.forceRefresh {
		if cached, err := h.db.GetPopularMessageDetails(job.messageID); err == nil && cached != nil {
			hasText := cached.Text != nil && *cached.Text != ""
			hasPermalink := cached.Permalink != nil && *cached.Permalink != ""
			hasAuthor := (cached.AuthorName != nil && *cached.AuthorName != "") || (cached.AuthorAvatar != nil && *cached.AuthorAvatar != "")
			imageKnown := cached.ImageURL != nil
			detailsFetchedKnown := cached.DetailsFetched != nil
			detailsFetched := detailsFetchedKnown && *cached.DetailsFetched
			if detailsFetched {
				return
			}
			if !detailsFetchedKnown && hasText && hasPermalink && hasAuthor && imageKnown {
				return
			}
		} else if err != nil {
			h.logger.Err(err).KV("message_id", job.messageID).Error("failed to read popular message cache before backfill")
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
			} else {
				h.notePopularBackfillFailure("search_author_timestamp_failed", job.messageID, err)
			}
		} else if err != nil {
			h.notePopularBackfillFailure("derive_author_failed", job.messageID, err)
		} else {
			h.notePopularBackfillFailure("author_missing", job.messageID, nil)
		}
		if channelID == "" {
			if found, err := h.slack.FindChannelByMessageID(job.messageID); err == nil {
				channelID = found
				if err := h.db.UpdateChannelIDForMessage(job.messageID, found); err != nil {
					h.logger.Err(err).KV("message_id", job.messageID).KV("channel_id", found).Error("failed to cache channel_id to database")
				}
			} else {
				h.notePopularBackfillFailure("search_timestamp_failed", job.messageID, err)
				return
			}
		}
	}

	details, err := h.slack.GetMessageDetails(channelID, job.messageID)
	if err != nil {
		h.notePopularBackfillFailure("get_message_details_failed", job.messageID, err)
		return
	}

	var text *string
	var authorID *string
	var authorName *string
	var authorAvatar *string
	var imageURL *string
	var attachmentURL *string
	var attachmentMime *string
	var reactionCount *int
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
		if details.AttachmentURL != "" {
			attachmentURL = &details.AttachmentURL
		}
		if details.AttachmentMime != "" {
			attachmentMime = &details.AttachmentMime
		}
		reactionCountVal := details.ReactionCount
		reactionCount = &reactionCountVal
		if details.AttachmentURL == "" && !details.HasAttachment {
			empty := ""
			attachmentURL = &empty
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
	detailsFetched := true
	if err := h.db.UpsertPopularMessageDetails(job.messageID, &channelID, text, permalink, authorID, authorName, authorAvatar, imageURL, attachmentURL, attachmentMime, reactionCount, isReply, isIgnored, &detailsFetched); err != nil {
		h.logger.Err(err).KV("message_id", job.messageID).Error("failed to cache popular message details")
	}
}
