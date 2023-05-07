# frozen_string_literal: true

class AddEmbeddingToStatus < ActiveRecord::Migration[6.1]
  def change
    add_column :statuses, :embedding, :float, array: true, default: [], null: true
  end
end
