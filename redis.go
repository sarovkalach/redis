package redis

import (
	"encoding/json"
	"errors"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/go-redis/redis"
)

const (
	defaultAddr = "0.0.0.0:6379" // "redis" name from docker-compose
	fastProxyDB = 2
	dbEmails    = 4
	tDomainsDB  = 5
)

const (
	debug = 1
)

var (
	errRedisConn = errors.New("Fail connect Redis")
	// errGetResult = errors.New("Error getting result from Redis")
)

var (
	addr = func() string {
		address := os.Getenv("REDIS_ADDR")
		if address == "" {
			return defaultAddr
		}
		return address
	}
)

type Proxy struct {
	Addr    string
	Counter int
}

type RedisEmail struct {
	Status    int8  `json:"validation_type"`
	Timestamp int32 `json:"timestamp"`
}

type TDomain struct {
	Timestamp int32 `json:"timestamp"`
}

func NewManager() *Manager {
	m := &Manager{}
	err := m.init()
	if err != nil {
		log.Fatal(err.Error(), addr())
	}
	return m
}

type Manager struct {
	tDomainsOptions  redis.Options
	tDomainsClient   redis.Client
	fastProxyOptions redis.Options
	FastProxyClient  redis.Client
	emailsOptions    redis.Options
	EmailsClient     redis.Client
	proxyList        []string
}

func (m *Manager) init() error {
	m.emailsOptions = redis.Options{
		Addr:     addr(),
		Password: "",       // no password set
		DB:       dbEmails, // use default DB
	}
	m.EmailsClient = *redis.NewClient(&m.emailsOptions)
	_, err := m.EmailsClient.Ping().Result()
	if err != nil {
		return errRedisConn
	}

	m.fastProxyOptions = redis.Options{
		Addr:     addr(),
		Password: "",          // no password set
		DB:       fastProxyDB, // use default DB
	}
	m.FastProxyClient = *redis.NewClient(&m.fastProxyOptions)

	m.tDomainsOptions = redis.Options{
		Addr:     addr(),
		Password: "",         // no password set
		DB:       tDomainsDB, // use default DB
	}
	m.tDomainsClient = *redis.NewClient(&m.tDomainsOptions)

	keys, _ := m.FastProxyClient.Keys("*").Result()
	m.proxyList = keys
	if debug > 0 {
		log.Println("Total Fast proxy count:", len(m.proxyList))
	}
	return nil
}

// FastProxy return random ip proxy from redis if attempts counter > 0
func (m *Manager) FastProxy() *Proxy {
	for range m.proxyList {
		rand.Seed(time.Now().UTC().UnixNano())
		addr := m.proxyList[rand.Intn(len(m.proxyList))]
		proxyCounter, err := m.FastProxyClient.Get(addr).Int()

		if err == nil && proxyCounter > 0 {
			m.FastProxyClient.Set(addr, proxyCounter, 0)
			return &Proxy{Addr: addr, Counter: proxyCounter}
		}
		continue
	}
	return nil
}

// SaveEmail save email in redis
func (m *Manager) SaveEmail(email string, status int8, lifetime int16) error {
	t, err := json.Marshal(RedisEmail{Status: status, Timestamp: int32(time.Now().Unix())})
	if err != nil {
		error1 := errors.New("Error in Marshalling result")
		return error1
	}
	if err := m.EmailsClient.Set(email, t, time.Duration(lifetime)*time.Hour).Err(); err != nil {
		return err
	}
	return nil
}

// CheckEmail return OK or BAD if email exist's in redis.
func (m *Manager) CheckEmail(email string) (int8, error) {
	res, err := m.EmailsClient.Get(email).Result()
	if err != nil {
		return -1, err
	}
	if res == "" {
		return -1, nil
	}
	var data RedisEmail
	res = strings.Replace(res, "'", "\"", -1)
	if err := json.Unmarshal([]byte(res), &data); err != nil {
		return -1, err
	}
	emailType := data.Status
	return emailType, nil
}

func (m *Manager) CheckTdomain(domain string) (bool, error) {
	res, _ := m.tDomainsClient.Get(domain).Result()

	if res == "" {
		return false, nil
	}
	return true, nil
}
