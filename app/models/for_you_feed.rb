# frozen_string_literal: true

class ForYouFeed < Feed
  def initialize(account)
    @account = account
    super(:for_you, account.id)
  end

  def regenerating?
    redis.exists?("account:#{@account.id}:regeneration")
  end
end
