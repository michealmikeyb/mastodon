# frozen_string_literal: true

class ForYouFeed < Feed
  def initialize(account)
    @account = account
    super(:for_you, account.id)
  end

  def regenerating?
    redis.exists?("account:#{@account.id}:regeneration")
  end

  def from_redis(limit, max_id, since_id, min_id)
    max_rank =  max_id.blank? ? '+inf' : redis.zrank(key, max_id)
    if min_id.blank?
      min_rank   = since_id.blank? ? '-inf' : redis.zrank(key, since_id)
      unhydrated = redis.zrevrangebyscore(key, "(#{max_rank}", "(#{min_rank}", limit: [0, limit], with_scores: true).map(&:first).map(&:to_i)
    else
      min_rank = redis.zrank(key, min_id)
      unhydrated = redis.zrangebyscore(key, "(#{min_rank}", "(#{max_rank}", limit: [0, limit], with_scores: true).map(&:first).map(&:to_i)
    end

    Status.where(id: unhydrated).cache_ids
  end
end
