package main

import (
	"fmt"
	"math/rand"
	"strings"
)

// URLGenerator generates realistic test URLs with various patterns.
type URLGenerator struct {
	rng *rand.Rand
}

// NewURLGenerator creates a new URL generator with the given seed.
func NewURLGenerator(seed int64) *URLGenerator {
	return &URLGenerator{
		rng: rand.New(rand.NewSource(seed)),
	}
}

// Generate produces a random URL with realistic patterns.
func (g *URLGenerator) Generate() string {
	patterns := []func() string{
		g.apiUserURL,
		g.apiProductURL,
		g.apiOrderURL,
		g.apiAnalyticsURL,
		g.apiPaymentURL,
		g.blogPostURL,
		g.documentURL,
		g.fileURL,
	}

	return patterns[g.rng.Intn(len(patterns))]()
}

// GenerateBatch produces n random URLs.
func (g *URLGenerator) GenerateBatch(n int) []string {
	urls := make([]string, n)
	for i := 0; i < n; i++ {
		urls[i] = g.Generate()
	}
	return urls
}

func (g *URLGenerator) uuid() string {
	const hex = "0123456789abcdef"
	var sb strings.Builder
	for i := 0; i < 32; i++ {
		if i == 8 || i == 12 || i == 16 || i == 20 {
			sb.WriteByte('-')
		}
		sb.WriteByte(hex[g.rng.Intn(16)])
	}
	return sb.String()
}

func (g *URLGenerator) numericID() string {
	return fmt.Sprintf("%d", 100000+g.rng.Intn(9900000))
}

func (g *URLGenerator) hash() string {
	const hex = "0123456789abcdef"
	var sb strings.Builder
	for i := 0; i < 24; i++ {
		sb.WriteByte(hex[g.rng.Intn(16)])
	}
	return sb.String()
}

func (g *URLGenerator) date() string {
	year := 2020 + g.rng.Intn(5)
	month := 1 + g.rng.Intn(12)
	day := 1 + g.rng.Intn(28)
	return fmt.Sprintf("%04d-%02d-%02d", year, month, day)
}

func (g *URLGenerator) timestamp() string {
	return fmt.Sprintf("%d", 1600000000+g.rng.Intn(100000000))
}

func (g *URLGenerator) slug() string {
	words := []string{"awesome", "cool", "new", "updated", "fresh", "best", "top", "great", "amazing", "super"}
	nouns := []string{"product", "post", "article", "item", "thing", "stuff", "deal", "offer", "review", "guide"}
	word1 := words[g.rng.Intn(len(words))]
	word2 := nouns[g.rng.Intn(len(nouns))]
	return fmt.Sprintf("%s-%s-%d", word1, word2, g.rng.Intn(99999))
}

func (g *URLGenerator) stripeID() string {
	prefixes := []string{"cus", "sub", "prod", "price", "pm", "pi", "ch"}
	prefix := prefixes[g.rng.Intn(len(prefixes))]
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	var sb strings.Builder
	for i := 0; i < 14; i++ {
		sb.WriteByte(chars[g.rng.Intn(len(chars))])
	}
	return prefix + "_" + sb.String()
}

func (g *URLGenerator) apiUserURL() string {
	actions := []string{"profile", "settings", "orders", "payments", "notifications", "preferences"}
	action := actions[g.rng.Intn(len(actions))]
	return fmt.Sprintf("/api/v1/users/%s/%s", g.uuid(), action)
}

func (g *URLGenerator) apiProductURL() string {
	actions := []string{"details", "reviews", "images", "variants", "inventory", "pricing"}
	action := actions[g.rng.Intn(len(actions))]
	return fmt.Sprintf("/api/v1/products/%s/%s", g.uuid(), action)
}

func (g *URLGenerator) apiOrderURL() string {
	actions := []string{"items", "status", "shipping", "tracking", "invoice", "refund"}
	action := actions[g.rng.Intn(len(actions))]
	return fmt.Sprintf("/api/v2/orders/%s/%s", g.numericID(), action)
}

func (g *URLGenerator) apiAnalyticsURL() string {
	metrics := []string{"pageviews", "sessions", "conversions", "revenue", "events"}
	metric := metrics[g.rng.Intn(len(metrics))]
	return fmt.Sprintf("/api/v1/analytics/%s/%s", g.date(), metric)
}

func (g *URLGenerator) apiPaymentURL() string {
	actions := []string{"capture", "refund", "details", "receipt"}
	action := actions[g.rng.Intn(len(actions))]
	return fmt.Sprintf("/api/v1/payments/%s/%s", g.stripeID(), action)
}

func (g *URLGenerator) blogPostURL() string {
	return fmt.Sprintf("/blog/%s", g.slug())
}

func (g *URLGenerator) documentURL() string {
	types := []string{"reports", "invoices", "contracts", "receipts"}
	docType := types[g.rng.Intn(len(types))]
	return fmt.Sprintf("/documents/%s/%s", docType, g.hash())
}

func (g *URLGenerator) fileURL() string {
	return fmt.Sprintf("/files/%s/download", g.hash())
}
