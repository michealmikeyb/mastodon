# frozen_string_literal: true

class Aggregate < ApplicationRecord
  belongs_to :account
  belongs_to :status
end
