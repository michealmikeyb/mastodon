# frozen_string_literal: true

class CreateAggregates < ActiveRecord::Migration[6.1]
  def change
    create_table :aggregates do |t|
      t.references :account, null: false, foreign_key: true
      t.references :status, null: false, foreign_key: true
      t.json :aggregate
      t.boolean :seen

      t.timestamps
    end
  end
end
