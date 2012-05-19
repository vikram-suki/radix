package redis

import (
	"flag"
	. "launchpad.net/gocheck"
	"testing"
	"time"
)

// Hook up gocheck into the gotest runner.
func Test(t *testing.T) {
	TestingT(t)
}

var rd *Client
var conf Configuration = Configuration{
	Database: 8,
	Path:     "/tmp/redis.sock",
	Timeout:  10,
}

type TI interface {
	Fatalf(string, ...interface{})
}

func setUpTest(c TI) {
	var err error

	rd, err = NewClient(conf)
	if err != nil {
		c.Fatalf("setUp NewClient failed: %s", err)
	}

	r := rd.Flushall()
	if r.Error != nil {
		c.Fatalf("setUp FLUSHALL failed: %s", r.Error)
	}
}

func tearDownTest(c TI) {
	r := rd.Flushall()
	if r.Error != nil {
		c.Fatalf("tearDown FLUSHALL failed: %s", r.Error)
	}

	rd.Close()
}

//* Tests
type S struct{}
type Long struct{}

var long = flag.Bool("long", false, "Include long running tests")

func init() {
	Suite(&S{})
	Suite(&Long{})
}

func (s *Long) SetUpSuite(c *C) {
	if !*long {
		c.Skip("-long not provided")
	}
}

func (s *S) SetUpTest(c *C) {
	setUpTest(c)
}

func (s *S) TearDownTest(c *C) {
	tearDownTest(c)
}

func (s *Long) SetUpTest(c *C) {
	setUpTest(c)
}

func (s *Long) TearDownTest(c *C) {
	tearDownTest(c)
}

// Test connection commands.
func (s *S) TestConnection(c *C) {
	c.Check(rd.Echo("Hello, World!").Str(), Equals, "Hello, World!")
	c.Check(rd.Ping().Str(), Equals, "PONG")
}

// Test single return value commands.
func (s *S) TestSimpleValue(c *C) {
	// Simple value commands.
	rd.Set("simple:string", "Hello,")
	rd.Append("simple:string", " World!")
	c.Check(rd.Get("simple:string").Str(), Equals, "Hello, World!")

	rd.Set("simple:int", 10)
	c.Check(rd.Incr("simple:int").Int(), Equals, 11)

	rd.Setbit("simple:bit", 0, true)
	rd.Setbit("simple:bit", 1, true)
	c.Check(rd.Getbit("simple:bit", 0).Bool(), Equals, true)
	c.Check(rd.Getbit("simple:bit", 1).Bool(), Equals, true)

	c.Check(rd.Get("non:existing:key").Nil(), Equals, true)
	c.Check(rd.Exists("non:existing:key").Bool(), Equals, false)
	c.Check(rd.Setnx("simple:nx", "Test").Bool(), Equals, true)
	c.Check(rd.Setnx("simple:nx", "Test").Bool(), Equals, false)
}

// Test multi return value commands.
func (s *S) TestMultiple(c *C) {
	// Set values first.
	rd.Set("multiple:a", "a")
	rd.Set("multiple:b", "b")
	rd.Set("multiple:c", "c")

	mulstr, err := rd.Mget("multiple:a", "multiple:b", "multiple:c").Strings()
	c.Assert(err, IsNil)
	c.Check(
		mulstr,
		DeepEquals,
		[]string{"a", "b", "c"},
	)
}

// Test hash accessing.
func (s *S) TestHash(c *C) {
	//* Single  return value commands.
	rd.Hset("hash:bool", "true:1", 1)
	rd.Hset("hash:bool", "true:2", true)
	rd.Hset("hash:bool", "true:3", "1")
	rd.Hset("hash:bool", "false:1", 0)
	rd.Hset("hash:bool", "false:2", false)
	rd.Hset("hash:bool", "false:3", "0")
	c.Check(rd.Hget("hash:bool", "true:1").Bool(), Equals, true)
	c.Check(rd.Hget("hash:bool", "true:2").Bool(), Equals, true)
	c.Check(rd.Hget("hash:bool", "true:3").Bool(), Equals, true)
	c.Check(rd.Hget("hash:bool", "false:1").Bool(), Equals, false)
	c.Check(rd.Hget("hash:bool", "false:2").Bool(), Equals, false)
	c.Check(rd.Hget("hash:bool", "false:3").Bool(), Equals, false)

	ha, err := rd.Hgetall("hash:bool").Map()
	c.Assert(err, IsNil)
	c.Check(ha["true:1"].Bool(), Equals, true)
	c.Check(ha["true:2"].Bool(), Equals, true)
	c.Check(ha["true:3"].Bool(), Equals, true)
	c.Check(ha["false:1"].Bool(), Equals, false)
	c.Check(ha["false:2"].Bool(), Equals, false)
	c.Check(ha["false:3"].Bool(), Equals, false)
}

// Test list commands.
func (s *S) TestList(c *C) {
	rd.Rpush("list:a", "one")
	rd.Rpush("list:a", "two")
	rd.Rpush("list:a", "three")
	rd.Rpush("list:a", "four")
	rd.Rpush("list:a", "five")
	rd.Rpush("list:a", "six")
	rd.Rpush("list:a", "seven")
	rd.Rpush("list:a", "eight")
	rd.Rpush("list:a", "nine")
	lranges, err := rd.Lrange("list:a", 0, -1).Strings()
	c.Assert(err, IsNil)
	c.Check(
		lranges,
		DeepEquals,
		[]string{"one", "two", "three", "four", "five", "six", "seven", "eight", "nine"})
	c.Check(rd.Lpop("list:a").Str(), Equals, "one")

	elems := rd.Lrange("list:a", 3, 6).Elems()
	c.Assert(len(elems), Equals, 4)
	c.Check(elems[0].Str(), Equals, "five")
	c.Check(elems[1].Str(), Equals, "six")
	c.Check(elems[2].Str(), Equals, "seven")
	c.Check(elems[3].Str(), Equals, "eight")

	rd.Ltrim("list:a", 0, 3)
	c.Check(rd.Llen("list:a").Int(), Equals, 4)

	rd.Rpoplpush("list:a", "list:b")
	c.Check(rd.Lindex("list:b", 4711).Nil(), Equals, true)
	c.Check(rd.Lindex("list:b", 0).Str(), Equals, "five")

	rd.Rpush("list:c", 1)
	rd.Rpush("list:c", 2)
	rd.Rpush("list:c", 3)
	rd.Rpush("list:c", 4)
	rd.Rpush("list:c", 5)
	c.Check(rd.Lpop("list:c").Str(), Equals, "1")

	lrangenil, err := rd.Lrange("non-existent-list", 0, -1).Strings()
	c.Assert(err, IsNil)
	c.Check(lrangenil, DeepEquals, []string{})
}

// Test set commands.
func (s *S) TestSets(c *C) {
	rd.Sadd("set:a", 1)
	rd.Sadd("set:a", 2)
	rd.Sadd("set:a", 3)
	rd.Sadd("set:a", 4)
	rd.Sadd("set:a", 5)
	rd.Sadd("set:a", 4)
	rd.Sadd("set:a", 3)
	c.Check(rd.Scard("set:a").Int(), Equals, 5)
	c.Check(rd.Sismember("set:a", "4").Bool(), Equals, true)
}

// Test argument formatting.
func (s *S) TestArgToRedis(c *C) {
	// string
	rd.Set("foo", "bar")
	c.Check(
		rd.Get("foo").Str(),
		Equals,
		"bar")

	// []byte
	rd.Set("foo2", []byte{'b', 'a', 'r'})
	c.Check(
		rd.Get("foo2").Bytes(),
		DeepEquals,
		[]byte{'b', 'a', 'r'})

	// bool
	rd.Set("foo3", true)
	c.Check(
		rd.Get("foo3").Bool(),
		Equals,
		true)

	// integers
	rd.Set("foo4", 2)
	c.Check(
		rd.Get("foo4").Str(),
		Equals,
		"2")

	// slice
	rd.Rpush("foo5", []int{1, 2, 3})
	foo5strings, err := rd.Lrange("foo5", 0, -1).Strings()
	c.Assert(err, IsNil)
	c.Check(
		foo5strings,
		DeepEquals,
		[]string{"1", "2", "3"})

	// map
	rd.Hset("foo6", "k1", "v1")
	rd.Hset("foo6", "k2", "v2")
	rd.Hset("foo6", "k3", "v3")

	foo6map, err := rd.Hgetall("foo6").StringMap()
	c.Assert(err, IsNil)
	c.Check(
		foo6map,
		DeepEquals,
		map[string]string{
			"k1": "v1",
			"k2": "v2",
			"k3": "v3",
		})
}

// Test asynchronous commands.
func (s *S) TestAsync(c *C) {
	fut := rd.AsyncPing()
	r := fut.Reply()
	c.Check(r.Str(), Equals, "PONG")
}

// Test multi-value commands.
func (s *S) TestMulti(c *C) {
	rd.Sadd("multi:set", "one")
	rd.Sadd("multi:set", "two")
	rd.Sadd("multi:set", "three")

	c.Check(rd.Smembers("multi:set").Len(), Equals, 3)
}

// Test multi commands.
func (s *S) TestMultiCommand(c *C) {
	r := rd.MultiCommand(func(mc *MultiCommand) {
		mc.Set("foo", "bar")
		mc.Get("foo")
	})
	c.Assert(r.Type, Equals, ReplyMulti)
	c.Check(r.At(0).Error, IsNil)
	c.Check(r.At(1).Str(), Equals, "bar")

	r = rd.MultiCommand(func(mc *MultiCommand) {
		mc.Set("foo2", "baz")
		mc.Get("foo2")
		rmc := mc.Flush()
		c.Check(rmc.At(0).Error, IsNil)
		c.Check(rmc.At(1).Str(), Equals, "baz")
		mc.Set("foo2", "qux")
		mc.Get("foo2")
	})
	c.Assert(r.Type, Equals, ReplyMulti)
	c.Check(r.At(0).Error, IsNil)
	c.Check(r.At(1).Str(), Equals, "qux")
}

// Test simple transactions.
func (s *S) TestTransaction(c *C) {
	r := rd.Transaction(func(mc *MultiCommand) {
		mc.Set("foo", "bar")
		mc.Get("foo")
	})
	c.Assert(r.Type, Equals, ReplyMulti)
	c.Check(r.At(0).Str(), Equals, "OK")
	c.Check(r.At(1).Str(), Equals, "bar")

	// Flushing transaction
	r = rd.Transaction(func(mc *MultiCommand) {
		mc.Set("foo", "bar")
		mc.Flush()
		mc.Get("foo")
	})
	c.Assert(r.Type, Equals, ReplyMulti)
	c.Check(r.Len(), Equals, 2)
	c.Check(r.At(0).Str(), Equals, "OK")
	c.Check(r.At(1).Str(), Equals, "bar")
}

// Test succesful complex tranactions.
func (s *S) TestComplexTransaction(c *C) {
	// Succesful transaction.
	r := rd.MultiCommand(func(mc *MultiCommand) {
		mc.Set("foo", "bar")
		mc.Watch("foo")
		rmc := mc.Flush()
		c.Assert(rmc.Type, Equals, ReplyMulti)
		c.Assert(rmc.Len(), Equals, 2)
		c.Assert(rmc.At(0).Error, IsNil)
		c.Assert(rmc.At(1).Error, IsNil)

		mc.Multi()
		mc.Set("foo", "baz")
		mc.Get("foo")
		mc.Command("brokenfunc")
		mc.Exec()
	})
	c.Assert(r.Type, Equals, ReplyMulti)
	c.Assert(r.Len(), Equals, 5)
	c.Check(r.At(0).Error, IsNil)
	c.Check(r.At(1).Error, IsNil)
	c.Check(r.At(2).Error, IsNil)
	c.Check(r.At(3).Error, NotNil)
	c.Assert(r.At(4).Type, Equals, ReplyMulti)
	c.Assert(r.At(4).Len(), Equals, 2)
	c.Check(r.At(4).At(0).Error, IsNil)
	c.Check(r.At(4).At(1).Str(), Equals, "baz")

	// Discarding transaction
	r = rd.MultiCommand(func(mc *MultiCommand) {
		mc.Set("foo", "bar")
		mc.Multi()
		mc.Set("foo", "baz")
		mc.Discard()
		mc.Get("foo")
	})
	c.Assert(r.Type, Equals, ReplyMulti)
	c.Assert(r.Len(), Equals, 5)
	c.Check(r.At(0).Error, IsNil)
	c.Check(r.At(1).Error, IsNil)
	c.Check(r.At(2).Error, IsNil)
	c.Check(r.At(3).Error, IsNil)
	c.Check(r.At(4).Error, IsNil)
	c.Check(r.At(4).Str(), Equals, "bar")
}

// Test asynchronous multi commands.
func (s *S) TestAsyncMultiCommand(c *C) {
	r := rd.AsyncMultiCommand(func(mc *MultiCommand) {
		mc.Set("foo", "bar")
		mc.Get("foo")
	}).Reply()
	c.Assert(r.Type, Equals, ReplyMulti)
	c.Check(r.At(0).Error, IsNil)
	c.Check(r.At(1).Str(), Equals, "bar")
}

// Test simple asynchronous transactions.
func (s *S) TestAsyncTransaction(c *C) {
	r := rd.AsyncTransaction(func(mc *MultiCommand) {
		mc.Set("foo", "bar")
		mc.Get("foo")
	}).Reply()
	c.Assert(r.Type, Equals, ReplyMulti)
	c.Check(r.At(0).Str(), Equals, "OK")
	c.Check(r.At(1).Str(), Equals, "bar")
}

// Test Subscription.
func (s *S) TestSubscription(c *C) {
	var messages []*Message
	msgHdlr := func(msg *Message) {
		c.Log(msg)
		messages = append(messages, msg)
	}

	sub, err := rd.Subscription(msgHdlr)
	if err != nil {
		c.Errorf("Failed to subscribe: '%v'!", err)
		return
	}

	sub.Subscribe("chan1", "chan2")

	c.Check(rd.Publish("chan1", "foo").Int(), Equals, 1)
	sub.Unsubscribe("chan1")
	c.Check(rd.Publish("chan1", "bar").Int(), Equals, 0)
	sub.Close()

	time.Sleep(time.Second)
	c.Assert(len(messages), Equals, 4)
	c.Check(messages[0].Type, Equals, MessageSubscribe)
	c.Check(messages[0].Channel, Equals, "chan1")
	c.Check(messages[0].Subscriptions, Equals, 1)
	c.Check(messages[1].Type, Equals, MessageSubscribe)
	c.Check(messages[1].Channel, Equals, "chan2")
	c.Check(messages[1].Subscriptions, Equals, 2)
	c.Check(messages[2].Type, Equals, MessageMessage)
	c.Check(messages[2].Channel, Equals, "chan1")
	c.Check(messages[2].Payload, Equals, "foo")
	c.Check(messages[3].Type, Equals, MessageUnsubscribe)
	c.Check(messages[3].Channel, Equals, "chan1")
}

// Test pattern subscriptions.
func (s *S) TestPSubscribe(c *C) {
	var messages []*Message
	msgHdlr := func(msg *Message) {
		c.Log(msg)
		messages = append(messages, msg)
	}

	sub, err := rd.Subscription(msgHdlr)
	if err != nil {
		c.Errorf("Failed to subscribe: '%v'!", err)
		return
	}

	sub.PSubscribe("foo.*")

	c.Check(rd.Publish("foo.foo", "foo").Int(), Equals, 1)
	sub.PUnsubscribe("foo.*")
	c.Check(rd.Publish("foo.bar", "bar").Int(), Equals, 0)
	sub.Close()

	time.Sleep(time.Second)
	c.Assert(len(messages), Equals, 3)
	c.Check(messages[0].Type, Equals, MessagePSubscribe)
	c.Check(messages[0].Pattern, Equals, "foo.*")
	c.Check(messages[0].Subscriptions, Equals, 1)
	c.Check(messages[1].Type, Equals, MessagePMessage)
	c.Check(messages[1].Channel, Equals, "foo.foo")
	c.Check(messages[1].Payload, Equals, "foo")
	c.Check(messages[1].Pattern, Equals, "foo.*")
	c.Check(messages[2].Type, Equals, MessagePUnsubscribe)
	c.Check(messages[2].Pattern, Equals, "foo.*")
}

// Test errors.
func (s *S) TestError(c *C) {
	err := newError("foo", ErrorConnection)
	c.Check(err.Error(), Equals, "redis: foo")
	c.Check(err.Test(ErrorConnection), Equals, true)
	c.Check(err.Test(ErrorRedis), Equals, false)

	errext := newErrorExt("bar", err, ErrorLoading)
	c.Check(errext.Error(), Equals, "redis: bar")
	c.Check(errext.Test(ErrorConnection), Equals, true)
	c.Check(errext.Test(ErrorLoading), Equals, true)
}

// Test tcp/ip connections.
func (s *S) TestTCP(c *C) {
	conf2 := conf
	conf2.Address = "127.0.0.1:6379"
	conf2.Path = ""
	rdA, errA := NewClient(conf2)
	c.Assert(errA, IsNil)
	rep := rdA.Echo("Hello, World!")
	c.Assert(rep.Error, IsNil)
	c.Check(rep.Str(), Equals, "Hello, World!")
}

// Test unix connections.
func (s *S) TestUnix(c *C) {
	conf2 := conf
	conf2.Address = ""
	conf2.Path = "/tmp/redis.sock"
	rdA, errA := NewClient(conf2)
	c.Assert(errA, IsNil)
	rep := rdA.Echo("Hello, World!")
	c.Assert(rep.Error, IsNil)
	c.Check(rep.Str(), Equals, "Hello, World!")
}

//* Long tests

// Test aborting complex tranactions.
func (s *Long) TestAbortingComplexTransaction(c *C) {
	go func() {
		time.Sleep(time.Second)
		rd.Set("foo", 9)
	}()

	r := rd.MultiCommand(func(mc *MultiCommand) {
		mc.Set("foo", 1)
		mc.Watch("foo")
		mc.Multi()
		rmc := mc.Flush()
		c.Assert(rmc.Type, Equals, ReplyMulti)
		c.Assert(rmc.Len(), Equals, 3)
		c.Assert(rmc.At(0).Error, IsNil)
		c.Assert(rmc.At(1).Error, IsNil)
		c.Assert(rmc.At(2).Error, IsNil)

		time.Sleep(time.Second * 2)
		mc.Set("foo", 2)
		mc.Exec()
	})
	c.Assert(r.Type, Equals, ReplyMulti)
	c.Assert(r.Len(), Equals, 2)
	c.Check(r.At(1).Nil(), Equals, true)
}

// Test timeout.
func (s *Long) TestTimeout(c *C) {
	conf2 := conf
	conf2.Path = ""
	conf2.Address = "127.0.0.1:12345"
	rdB, errB := NewClient(conf2)
	c.Assert(errB, IsNil)
	rB := rdB.Ping()
	c.Log(rB.Error)
	c.Check(rB.Error, NotNil)
}

// Test illegal database.
func (s *Long) TestIllegalDatabase(c *C) {
	conf2 := conf
	conf2.Database = 4711
	rdA, errA := NewClient(conf2)
	c.Assert(errA, IsNil)
	rA := rdA.Ping()
	c.Check(rA.Error, NotNil)
}

//* Benchmarks

func BenchmarkBlockingPing(b *testing.B) {
	setUpTest(b)

	for i := 0; i < b.N; i++ {
		rd.Ping()
	}

	tearDownTest(b)
}

func BenchmarkBlockingSet(b *testing.B) {
	setUpTest(b)

	for i := 0; i < b.N; i++ {
		rd.Set("foo", "bar")
	}

	tearDownTest(b)
}

func BenchmarkBlockingGet(b *testing.B) {
	setUpTest(b)

	for i := 0; i < b.N; i++ {
		rd.Get("foo", "bar")
	}

	tearDownTest(b)
}

func BenchmarkAsyncPing(b *testing.B) {
	setUpTest(b)

	for i := 0; i < b.N; i++ {
		fut := rd.AsyncPing()
		fut.Reply()
	}

	tearDownTest(b)
}

func BenchmarkAsyncSet(b *testing.B) {
	setUpTest(b)

	for i := 0; i < b.N; i++ {
		fut := rd.AsyncSet("foo", "bar")
		fut.Reply()
	}

	tearDownTest(b)
}

func BenchmarkAsyncGet(b *testing.B) {
	setUpTest(b)

	for i := 0; i < b.N; i++ {
		fut := rd.AsyncGet("foo", "bar")
		fut.Reply()
	}

	tearDownTest(b)
}

func BenchmarkConnectionPool(b *testing.B) {
	setUpTest(b)

	for i := 0; i < b.N; i++ {
		fut := rd.AsyncGet("foo", "bar")
		fut.Reply()
	}

	tearDownTest(b)
}