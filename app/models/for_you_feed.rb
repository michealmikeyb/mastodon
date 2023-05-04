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
    max_score = max_id.blank? ? '+inf' : redis.zscore(key, max_id)
    if min_id.blank?
      min_score = since_id.blank? ? '-inf' : redis.zscore(key, since_id)
      unhydrated = redis.zrevrangebyscore(key, "(#{max_score}", "(#{min_score}", limit: [0, limit], with_scores: true).map(&:first).map(&:to_i)
    else
      min_score = redis.zscore(key, min_id)
      unhydrated = redis.zrangebyscore(key, "(#{min_score}", "(#{max_score}", limit: [0, limit], with_scores: true).map(&:first).map(&:to_i)
    end
    unordered_statuses = Status.where(id: unhydrated).cache_ids
    ordered_statuses = []
    unhydrated.each do |status_id|
      unordered_statuses.each do |status|
        ordered_statuses.append(status) if status.id == status_id
      end
    end
    ordered_statuses
  end
end
