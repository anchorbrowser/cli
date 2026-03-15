package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

func (c *Client) SessionCreate(ctx context.Context, apiKey string, req any) (any, error) {
	return c.JSON(ctx, apiKey, http.MethodPost, "/v1/sessions", nil, req)
}

func (c *Client) SessionList(ctx context.Context, apiKey string, query url.Values) (any, error) {
	return c.JSON(ctx, apiKey, http.MethodGet, "/v1/sessions", query, nil)
}

func (c *Client) SessionGet(ctx context.Context, apiKey, sessionID string) (any, error) {
	return c.JSON(ctx, apiKey, http.MethodGet, "/v1/sessions/"+url.PathEscape(sessionID), nil, nil)
}

func (c *Client) SessionEnd(ctx context.Context, apiKey, sessionID string) (any, error) {
	return c.JSON(ctx, apiKey, http.MethodDelete, "/v1/sessions/"+url.PathEscape(sessionID), nil, nil)
}

func (c *Client) SessionEndAll(ctx context.Context, apiKey string) (any, error) {
	return c.JSON(ctx, apiKey, http.MethodDelete, "/v1/sessions/all", nil, nil)
}

func (c *Client) SessionPages(ctx context.Context, apiKey, sessionID string) (any, error) {
	return c.JSON(ctx, apiKey, http.MethodGet, "/v1/sessions/"+url.PathEscape(sessionID)+"/pages", nil, nil)
}

func (c *Client) SessionHistory(ctx context.Context, apiKey string, query url.Values) (any, error) {
	return c.JSON(ctx, apiKey, http.MethodGet, "/v1/sessions/history", query, nil)
}

func (c *Client) SessionStatusAll(ctx context.Context, apiKey string, query url.Values) (any, error) {
	return c.JSON(ctx, apiKey, http.MethodGet, "/v1/sessions/all/status", query, nil)
}

func (c *Client) SessionDownloads(ctx context.Context, apiKey, sessionID string) (any, error) {
	return c.JSON(ctx, apiKey, http.MethodGet, "/v1/sessions/"+url.PathEscape(sessionID)+"/downloads", nil, nil)
}

func (c *Client) SessionRecordings(ctx context.Context, apiKey, sessionID string) (any, error) {
	return c.JSON(ctx, apiKey, http.MethodGet, "/v1/sessions/"+url.PathEscape(sessionID)+"/recordings", nil, nil)
}

func (c *Client) SessionRecordingFetchPrimary(ctx context.Context, apiKey, sessionID string) ([]byte, error) {
	return c.Binary(ctx, apiKey, http.MethodGet, "/v1/sessions/"+url.PathEscape(sessionID)+"/recordings/primary/fetch", nil, nil)
}

func (c *Client) SessionScreenshot(ctx context.Context, apiKey, sessionID string) ([]byte, error) {
	return c.Binary(ctx, apiKey, http.MethodGet, "/v1/sessions/"+url.PathEscape(sessionID)+"/screenshot", nil, nil)
}

func (c *Client) SessionClick(ctx context.Context, apiKey, sessionID string, body any) (any, error) {
	return c.JSON(ctx, apiKey, http.MethodPost, "/v1/sessions/"+url.PathEscape(sessionID)+"/mouse/click", nil, body)
}

func (c *Client) SessionDoubleClick(ctx context.Context, apiKey, sessionID string, body any) (any, error) {
	return c.JSON(ctx, apiKey, http.MethodPost, "/v1/sessions/"+url.PathEscape(sessionID)+"/mouse/doubleClick", nil, body)
}

func (c *Client) SessionMouseDown(ctx context.Context, apiKey, sessionID string, body any) (any, error) {
	return c.JSON(ctx, apiKey, http.MethodPost, "/v1/sessions/"+url.PathEscape(sessionID)+"/mouse/down", nil, body)
}

func (c *Client) SessionMouseUp(ctx context.Context, apiKey, sessionID string, body any) (any, error) {
	return c.JSON(ctx, apiKey, http.MethodPost, "/v1/sessions/"+url.PathEscape(sessionID)+"/mouse/up", nil, body)
}

func (c *Client) SessionMove(ctx context.Context, apiKey, sessionID string, body any) (any, error) {
	return c.JSON(ctx, apiKey, http.MethodPost, "/v1/sessions/"+url.PathEscape(sessionID)+"/mouse/move", nil, body)
}

func (c *Client) SessionDragDrop(ctx context.Context, apiKey, sessionID string, body any) (any, error) {
	return c.JSON(ctx, apiKey, http.MethodPost, "/v1/sessions/"+url.PathEscape(sessionID)+"/drag-and-drop", nil, body)
}

func (c *Client) SessionScroll(ctx context.Context, apiKey, sessionID string, body any) (any, error) {
	return c.JSON(ctx, apiKey, http.MethodPost, "/v1/sessions/"+url.PathEscape(sessionID)+"/scroll", nil, body)
}

func (c *Client) SessionType(ctx context.Context, apiKey, sessionID string, body any) (any, error) {
	return c.JSON(ctx, apiKey, http.MethodPost, "/v1/sessions/"+url.PathEscape(sessionID)+"/keyboard/type", nil, body)
}

func (c *Client) SessionShortcut(ctx context.Context, apiKey, sessionID string, body any) (any, error) {
	return c.JSON(ctx, apiKey, http.MethodPost, "/v1/sessions/"+url.PathEscape(sessionID)+"/keyboard/shortcut", nil, body)
}

func (c *Client) SessionClipboardGet(ctx context.Context, apiKey, sessionID string) (any, error) {
	return c.JSON(ctx, apiKey, http.MethodGet, "/v1/sessions/"+url.PathEscape(sessionID)+"/clipboard", nil, nil)
}

func (c *Client) SessionClipboardSet(ctx context.Context, apiKey, sessionID string, body any) (any, error) {
	return c.JSON(ctx, apiKey, http.MethodPost, "/v1/sessions/"+url.PathEscape(sessionID)+"/clipboard", nil, body)
}

func (c *Client) SessionCopy(ctx context.Context, apiKey, sessionID string) (any, error) {
	return c.JSON(ctx, apiKey, http.MethodPost, "/v1/sessions/"+url.PathEscape(sessionID)+"/copy", nil, nil)
}

func (c *Client) SessionPaste(ctx context.Context, apiKey, sessionID string, body any) (any, error) {
	return c.JSON(ctx, apiKey, http.MethodPost, "/v1/sessions/"+url.PathEscape(sessionID)+"/paste", nil, body)
}

func (c *Client) SessionGoto(ctx context.Context, apiKey, sessionID string, body any) (any, error) {
	return c.JSON(ctx, apiKey, http.MethodPost, "/v1/sessions/"+url.PathEscape(sessionID)+"/goto", nil, body)
}

func (c *Client) SessionUpload(ctx context.Context, apiKey, sessionID, filePath string) (any, error) {
	return c.MultipartFile(ctx, apiKey, http.MethodPost, "/v1/sessions/"+url.PathEscape(sessionID)+"/uploads", nil, "file", filePath)
}

func (c *Client) AgentRun(ctx context.Context, apiKey string, sessionID string, body any) (any, error) {
	query := url.Values{}
	if sessionID != "" {
		query.Set("sessionId", sessionID)
	}
	return c.JSON(ctx, apiKey, http.MethodPost, "/v1/tools/perform-web-task", query, body)
}

func (c *Client) AgentRunStatus(ctx context.Context, apiKey, workflowID string) (any, error) {
	return c.JSON(ctx, apiKey, http.MethodGet, "/v1/tools/perform-web-task/"+url.PathEscape(workflowID)+"/status", nil, nil)
}

func (c *Client) TaskRun(ctx context.Context, apiKey, taskID string, body any) (any, error) {
	return c.JSON(ctx, apiKey, http.MethodPost, "/v2/tasks/"+url.PathEscape(taskID)+"/run", nil, body)
}

func (c *Client) TaskStatus(ctx context.Context, apiKey, runID string) (any, error) {
	return c.JSON(ctx, apiKey, http.MethodGet, "/v2/tasks/runs/"+url.PathEscape(runID)+"/status", nil, nil)
}

func (c *Client) IdentityCreate(ctx context.Context, apiKey string, validateAsync bool, body any) (any, error) {
	query := url.Values{}
	query.Set("validateAsync", fmt.Sprintf("%t", validateAsync))
	return c.JSON(ctx, apiKey, http.MethodPost, "/v1/identities", query, body)
}

func (c *Client) IdentityGet(ctx context.Context, apiKey, identityID string) (any, error) {
	return c.JSON(ctx, apiKey, http.MethodGet, "/v1/identities/"+url.PathEscape(identityID), nil, nil)
}

func (c *Client) IdentityUpdate(ctx context.Context, apiKey, identityID string, body any) (any, error) {
	return c.JSON(ctx, apiKey, http.MethodPut, "/v1/identities/"+url.PathEscape(identityID), nil, body)
}

func (c *Client) IdentityDelete(ctx context.Context, apiKey, identityID string) (any, error) {
	return c.JSON(ctx, apiKey, http.MethodDelete, "/v1/identities/"+url.PathEscape(identityID), nil, nil)
}

func (c *Client) IdentityCredentials(ctx context.Context, apiKey, identityID string) (any, error) {
	return c.JSON(ctx, apiKey, http.MethodGet, "/v1/identities/"+url.PathEscape(identityID)+"/credentials", nil, nil)
}

func (c *Client) ApplicationList(ctx context.Context, apiKey string, query url.Values) (any, error) {
	return c.JSON(ctx, apiKey, http.MethodGet, "/v1/applications", query, nil)
}

func (c *Client) ApplicationListIdentities(ctx context.Context, apiKey, applicationID string, query url.Values) (any, error) {
	return c.JSON(ctx, apiKey, http.MethodGet, "/v1/applications/"+url.PathEscape(applicationID)+"/identities", query, nil)
}
