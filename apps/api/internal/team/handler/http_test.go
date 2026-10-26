package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"teamflow/internal/httpx"
	"teamflow/internal/team"

	"github.com/gin-gonic/gin"
)

type fakeTeamHandler struct {
	created *CreateTeamReq
	resp    *CreateTeamResp
}

func (f *fakeTeamHandler) CreateTeam(_ context.Context, req *CreateTeamReq) (*CreateTeamResp, error) {
	f.created = req
	return f.resp, nil
}
func (f *fakeTeamHandler) ListTeams(context.Context, *ListTeamsReq) (*ListTeamsResp, error) {
	return &ListTeamsResp{}, nil
}
func (f *fakeTeamHandler) GetTeam(context.Context, *GetTeamReq) (*GetTeamResp, error) {
	return nil, nil
}
func (f *fakeTeamHandler) UpdateTeam(context.Context, *UpdateTeamReq) (*UpdateTeamResp, error) {
	return nil, nil
}
func (f *fakeTeamHandler) DeleteTeam(context.Context, *DeleteTeamReq) (*DeleteTeamResp, error) {
	return nil, nil
}

func TestRegisterRoutesInjectVerifiedActor(t *testing.T) {
	gin.SetMode(gin.TestMode)
	app := &fakeTeamHandler{resp: &CreateTeamResp{Team: team.Team{Name: "core"}}}
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(httpx.ActorIDKey, uint(9))
		c.Next()
	})
	api := router.Group("/api/v1")
	New(app).Register(api)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/teams", strings.NewReader(`{"name":"core","actor_id":42}`))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s, want 201", response.Code, response.Body.String())
	}
	if app.created == nil || app.created.ActorID != 9 {
		t.Fatalf("actor=%+v, want verified actor 9 (client body must not override)", app.created)
	}
}

func TestRegisterRejectsInvalidPathID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	app := &fakeTeamHandler{}
	router := gin.New()
	api := router.Group("/api/v1")
	New(app).Register(api)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/teams/abc", nil))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400", response.Code)
	}
}
