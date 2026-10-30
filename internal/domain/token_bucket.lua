local key = KEYS[1]

local capacity = tonumber(ARGV[1])
local refill_rate = tonumber(ARGV[2])
local t = redis.call('TIME')
local now = tonumber(t[1]) + tonumber(t[2]) / 1000000

local bucket = redis.call('HMGET', key, 'tokens', 'last_refill')
local tokens = tonumber(bucket[1])
local last_refill = tonumber(bucket[2])

if tokens == nil then
    tokens = capacity
    last_refill = now
end

local time_passed = now - last_refill

if time_passed > 0 then
    tokens = math.min(capacity, tokens + (time_passed * refill_rate))
    last_refill = now
end

local allowed = 0
if tokens >= 1 then
    tokens = tokens - 1
    allowed = 1
end

redis.call('HSET', key, 'tokens', tokens, 'last_refill', last_refill)

local missing = capacity - tokens
local tofull = missing / refill_rate
redis.call('EXPIRE', key, math.ceil(tofull) + 1)

return {allowed, tokens}
