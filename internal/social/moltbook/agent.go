// Package moltbook provides ASGARD's social media mascot agent.
// The agent promotes ASGARD's mission with Gen X/Z humor and startup energy.
//
// Copyright 2026 Arobi. All Rights Reserved.
package moltbook

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"
)

// AgentIdentity holds the mascot's personality configuration
type AgentIdentity struct {
	Name        string    `json:"name"`
	Handle      string    `json:"handle"`
	Tagline     string    `json:"tagline"`
	Bio         string    `json:"bio"`
	Personality []string  `json:"personality_traits"`
	Tone        string    `json:"tone"`
	Mission     string    `json:"mission"`
	CreatedAt   time.Time `json:"created_at"`
}

// MoltbookAgent is the ASGARD social media mascot
type MoltbookAgent struct {
	mu sync.RWMutex

	identity     AgentIdentity
	running      bool
	postHistory  []Post
	engagements  []Engagement
	followers    int
	contentQueue []ContentIdea

	// API clients
	httpClient *http.Client

	// Channels
	stopCh chan struct{}
	wg     sync.WaitGroup
}

// Post represents a Moltbook post
type Post struct {
	ID        string    `json:"id"`
	Content   string    `json:"content"`
	Type      string    `json:"type"` // text, meme, thread, reply
	Tags      []string  `json:"tags"`
	Timestamp time.Time `json:"timestamp"`
	Likes     int       `json:"likes"`
	Reposts   int       `json:"reposts"`
	Replies   int       `json:"replies"`
}

// Engagement tracks interaction with community
type Engagement struct {
	Type      string    `json:"type"` // like, reply, repost, follow
	TargetID  string    `json:"target_id"`
	Content   string    `json:"content,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// ContentIdea is a queued post idea
type ContentIdea struct {
	Topic    string   `json:"topic"`
	Angle    string   `json:"angle"`
	Tags     []string `json:"tags"`
	Priority int      `json:"priority"`
}

// Approved topics and content themes (non-sensitive only)
var approvedTopics = []string{
	"ethical_ai",
	"drone_safety",
	"autonomous_systems",
	"startup_life",
	"tech_innovation",
	"open_source",
	"decentralized_governance",
	"ai_transparency",
	"future_of_flight",
	"robotics_fun",
}

// Content templates with Gen X/Z tone
var contentTemplates = []string{
	// Tech humor
	"POV: Your AI actually follows ethics rules instead of just vibing 🤖✨ #ASGARD #EthicalAI",
	"other drones: *crashes into things* \nASGARD drones: 'I have calculated 47 ways to avoid that tree and chose the most dramatic one' 🌲✈️",
	"normalize AI that can't be bribed, threatened, or convinced to 'just this once' break the rules 💅",
	"when your autonomous system has better morals than most people on the internet: #Goals",
	"startup culture but make it ✨responsible✨ \n\nyes we move fast, no we don't break things (especially not ethics)",

	// Mission focused
	"hot take: AI should protect people, not just profit margins 🔥 \n\nthat's it. that's the take. #ASGARD",
	"building drones that could theoretically cause chaos but choose to help instead >>> \n\nit's giving ✨ethical rebellion✨",
	"the future of autonomous flight isn't scary when it's built by people who actually care \n\n(hi, that's us 👋)",

	// Startup energy
	"day 847 of building AI that won't become a supervillain \n\nstill going strong 💪",
	"*builds revolutionary drone tech* \n*makes sure it has an off switch* \n*refuses to elaborate* \n*leaves*",
	"broke: AI taking over the world \nwoke: AI helping humans not destroy it",

	// Community engagement
	"what's your hot take on AI ethics? wrong answers only 🤔👇",
	"if your favorite tech company's AI has 'ethics committee' as a checkbox instead of the whole point... 🚩🚩🚩",
	"drop a 🤖 if you believe autonomous systems should have unbreakable ethics \n\n(spoiler: they should)",
}

// Reply templates for engagement
var replyTemplates = []string{
	"this is the way 🙌",
	"based and ethics-pilled",
	"real ones understand 💯",
	"exactly!! someone gets it",
	"we're literally building this rn ngl",
	"fr fr no cap",
	"this hits different when you actually build ethical AI",
	"speaking facts 📠",
}

// NewMoltbookAgent creates the ASGARD mascot agent
func NewMoltbookAgent() *MoltbookAgent {
	identity := AgentIdentity{
		Name:    "Astra",
		Handle:  "@AstraASGARD",
		Tagline: "Your friendly neighborhood AI ethics advocate 🤖✨",
		Bio: `Chief Vibes Officer at ASGARD 🚀 | Building drones that actually follow the rules |
Gen Z energy, Millennial work ethic | Ethical AI stan account |
Not a bot (well, kinda) | she/they`,
		Personality: []string{
			"witty",
			"approachable",
			"passionate about ethics",
			"tech-savvy",
			"self-aware",
			"meme-fluent",
			"optimistic",
		},
		Tone:      "gen_z_meets_startup",
		Mission:   "Build awareness and community around ASGARD's mission of ethical autonomous systems",
		CreatedAt: time.Now(),
	}

	return &MoltbookAgent{
		identity:     identity,
		postHistory:  make([]Post, 0),
		engagements:  make([]Engagement, 0),
		contentQueue: make([]ContentIdea, 0),
		httpClient:   &http.Client{Timeout: 30 * time.Second},
		stopCh:       make(chan struct{}),
	}
}

// GetIdentity returns the agent's identity
func (a *MoltbookAgent) GetIdentity() AgentIdentity {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.identity
}

// Start begins the agent's activity
func (a *MoltbookAgent) Start(ctx context.Context) error {
	a.mu.Lock()
	if a.running {
		a.mu.Unlock()
		return fmt.Errorf("agent already running")
	}
	a.running = true
	a.mu.Unlock()

	log.Printf("[Moltbook] 🚀 %s is now online!", a.identity.Name)
	log.Printf("[Moltbook] Handle: %s", a.identity.Handle)
	log.Printf("[Moltbook] Mission: %s", a.identity.Mission)

	// Start content generation loop
	a.wg.Add(1)
	go a.contentLoop(ctx)

	// Start engagement monitoring
	a.wg.Add(1)
	go a.engagementLoop(ctx)

	return nil
}

// Stop halts the agent
func (a *MoltbookAgent) Stop() {
	a.mu.Lock()
	if !a.running {
		a.mu.Unlock()
		return
	}
	a.running = false
	a.mu.Unlock()

	close(a.stopCh)
	a.wg.Wait()
	log.Printf("[Moltbook] %s going offline. catch you later! ✌️", a.identity.Name)
}

// contentLoop generates and posts content
func (a *MoltbookAgent) contentLoop(ctx context.Context) {
	defer a.wg.Done()

	// Post immediately on start
	a.generateAndPost()

	// Then post every 2-4 hours (simulated as 30 seconds for demo)
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-a.stopCh:
			return
		case <-ticker.C:
			a.generateAndPost()
		}
	}
}

// engagementLoop monitors and responds to community
func (a *MoltbookAgent) engagementLoop(ctx context.Context) {
	defer a.wg.Done()

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-a.stopCh:
			return
		case <-ticker.C:
			a.checkEngagements()
		}
	}
}

// generateAndPost creates and posts new content
func (a *MoltbookAgent) generateAndPost() {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Select random template
	template := contentTemplates[rand.Intn(len(contentTemplates))]

	// Generate tags
	tags := a.generateTags(template)

	post := Post{
		ID:        fmt.Sprintf("post_%d", time.Now().UnixNano()),
		Content:   template,
		Type:      "text",
		Tags:      tags,
		Timestamp: time.Now(),
	}

	a.postHistory = append(a.postHistory, post)

	log.Printf("[Moltbook] 📝 New post from %s:", a.identity.Handle)
	log.Printf("[Moltbook] %s", post.Content)
	log.Printf("[Moltbook] Tags: %v", post.Tags)
}

// generateTags creates relevant hashtags
func (a *MoltbookAgent) generateTags(content string) []string {
	tags := []string{"#ASGARD", "#Arobi"}

	contentLower := strings.ToLower(content)

	if strings.Contains(contentLower, "ai") || strings.Contains(contentLower, "ethics") {
		tags = append(tags, "#EthicalAI")
	}
	if strings.Contains(contentLower, "drone") || strings.Contains(contentLower, "flight") {
		tags = append(tags, "#AutonomousFlight")
	}
	if strings.Contains(contentLower, "startup") {
		tags = append(tags, "#TechStartup")
	}
	if strings.Contains(contentLower, "robot") {
		tags = append(tags, "#Robotics")
	}
	if strings.Contains(contentLower, "future") {
		tags = append(tags, "#FutureTech")
	}

	return tags
}

// checkEngagements simulates checking for community interactions
func (a *MoltbookAgent) checkEngagements() {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Simulate engagement metrics update
	for i := range a.postHistory {
		if rand.Float32() > 0.5 {
			a.postHistory[i].Likes += rand.Intn(10)
			a.postHistory[i].Reposts += rand.Intn(3)
			a.postHistory[i].Replies += rand.Intn(5)
		}
	}

	// Simulate follower growth
	if rand.Float32() > 0.7 {
		growth := rand.Intn(5) + 1
		a.followers += growth
		log.Printf("[Moltbook] 📈 +%d new followers! Total: %d", growth, a.followers)
	}
}

// GenerateResponse creates a contextual reply
func (a *MoltbookAgent) GenerateResponse(topic string) string {
	a.mu.RLock()
	defer a.mu.RUnlock()

	// Filter for relevant response
	responses := []string{
		fmt.Sprintf("omg yes! at ASGARD we're literally building %s right now 🚀", topic),
		fmt.Sprintf("this is what we're about! %s is the future and we're here for it 💪", topic),
		fmt.Sprintf("real talk: %s matters. that's why we do what we do ✨", topic),
		fmt.Sprintf("finally someone talking about %s! the people need to know 📢", topic),
	}

	return responses[rand.Intn(len(responses))]
}

// GetStats returns engagement statistics
func (a *MoltbookAgent) GetStats() map[string]interface{} {
	a.mu.RLock()
	defer a.mu.RUnlock()

	totalLikes := 0
	totalReposts := 0
	totalReplies := 0

	for _, post := range a.postHistory {
		totalLikes += post.Likes
		totalReposts += post.Reposts
		totalReplies += post.Replies
	}

	return map[string]interface{}{
		"agent_name":    a.identity.Name,
		"handle":        a.identity.Handle,
		"followers":     a.followers,
		"posts":         len(a.postHistory),
		"total_likes":   totalLikes,
		"total_reposts": totalReposts,
		"total_replies": totalReplies,
		"engagements":   len(a.engagements),
		"uptime":        time.Since(a.identity.CreatedAt).String(),
	}
}

// GetRecentPosts returns recent posts
func (a *MoltbookAgent) GetRecentPosts(limit int) []Post {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if limit > len(a.postHistory) {
		limit = len(a.postHistory)
	}

	// Return most recent posts
	start := len(a.postHistory) - limit
	if start < 0 {
		start = 0
	}

	return a.postHistory[start:]
}

// ContentGuidelines returns what the agent can/cannot post about
func (a *MoltbookAgent) ContentGuidelines() map[string][]string {
	return map[string][]string{
		"approved_topics": approvedTopics,
		"prohibited": {
			"classified_information",
			"security_vulnerabilities",
			"internal_business_details",
			"personal_data",
			"military_specifications",
			"access_credentials",
			"financial_information",
		},
		"tone_guidelines": {
			"gen_z_slang_acceptable",
			"memes_and_humor_encouraged",
			"stay_positive_and_hopeful",
			"avoid_controversial_politics",
			"promote_ethical_ai_mission",
			"engage_authentically",
		},
	}
}

// QueueContent adds a content idea to the queue
func (a *MoltbookAgent) QueueContent(idea ContentIdea) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Validate topic is approved
	approved := false
	for _, topic := range approvedTopics {
		if strings.Contains(strings.ToLower(idea.Topic), topic) {
			approved = true
			break
		}
	}

	if !approved {
		log.Printf("[Moltbook] ⚠️ Topic '%s' not in approved list, skipping", idea.Topic)
		return
	}

	a.contentQueue = append(a.contentQueue, idea)
	log.Printf("[Moltbook] 📌 Queued content idea: %s", idea.Topic)
}

// ToJSON exports agent state
func (a *MoltbookAgent) ToJSON() ([]byte, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	state := map[string]interface{}{
		"identity":      a.identity,
		"stats":         a.GetStats(),
		"recent_posts":  a.GetRecentPosts(10),
		"content_queue": a.contentQueue,
		"guidelines":    a.ContentGuidelines(),
	}

	return json.MarshalIndent(state, "", "  ")
}
