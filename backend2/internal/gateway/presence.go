package gateway

import (
	"backend/internal/protocol"
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

var renewPresenceScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
    return redis.call("PEXPIRE", KEYS[1], ARGV[2])
end
return 0
`)

var releasePresenceScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
    return redis.call("DEL", KEYS[1])
end
return 0
`)

func (g *Gateway) presenceToken(c *Client) string {
	return g.cfg.GatewayID + ":" + c.ID
}

func (g *Gateway) reservePresence(ctx context.Context, c *Client, username string) (bool, error) {
	return g.redis.SetNX(
		ctx,
		protocol.OnlineUserKey(username),
		g.presenceToken(c),
		g.cfg.PresenceTTL,
	).Result()
}

func (g *Gateway) releasePresence(ctx context.Context, c *Client, username string) error {
	if username == "" {
		return nil
	}
	_, err := releasePresenceScript.Run(
		ctx,
		g.redis,
		[]string{protocol.OnlineUserKey(username)},
		g.presenceToken(c),
	).Result()
	if errors.Is(err, redis.Nil) {
		return nil
	}
	return err
}

func (g *Gateway) presenceLoop(ctx context.Context) {
	ticker := time.NewTicker(g.cfg.PresenceRefresh)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for _, c := range g.clients.All() {
				s := c.snapshot()
				if !s.IsAuthenticated || s.Username == "" {
					continue
				}

				refreshCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
				result, err := renewPresenceScript.Run(
					refreshCtx,
					g.redis,
					[]string{protocol.OnlineUserKey(s.Username)},
					g.presenceToken(c),
					fmt.Sprint(g.cfg.PresenceTTL.Milliseconds()),
				).Int()
				cancel()
				if err != nil && !errors.Is(err, redis.Nil) {
					log.Printf("gateway: refresh presence for %s: %v", s.Username, err)
					continue
				}
				if result == 0 {
					log.Printf("gateway: presence ownership for %s was lost", s.Username)
				}
			}
		}
	}
}
