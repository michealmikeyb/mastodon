# frozen_string_literal: true

require 'net/http'
require 'json'

class Api::V1::Trends::StatusesController < Api::BaseController
  vary_by 'Authorization, Accept-Language'

  before_action :set_statuses

  after_action :insert_pagination_headers

  def index
    cache_if_unauthenticated!
    render json: @statuses, each_serializer: REST::StatusSerializer
  end

  private

  def enabled?
    Setting.trends
  end

  def fetch_external_trends
    statuses = []
    uri_string = "https://sfba.social/api/v1/trends/statuses?limit=#{limit_param(20)}&offset=#{offset_param}"
    uri = URI(uri_string)
    res = Net::HTTP.get_response(uri)
    res_json = JSON.parse(res.body)
    res_json.each do |status_json|
      status = FetchRemoteStatusService.new.call(status_json['url'])
      statuses.append(status) unless status.nil?
    end
    # uri_string = 'https://lemmy.world/api/v3/post/list?sort=Active&limit=25'
    # uri = URI(uri_string)
    # res = Net::HTTP.get_response(uri)
    # res_json = JSON.parse(res.body)
    # res_json['posts'].each do |status_json|
    #   status = FetchRemoteStatusService.new.call(status_json['post']['ap_id'])
    #   if !status_json['post']['body'].nil? && !status.nil?
    #     status.text = status_json['post']['name'] + "<p>body: #{status_json['post']['body']}</p>"
    #     status.save!
    #   end
    #   if !status_json['post']['url'].nil? && !status.nil? && status_json['post']['url'].end_with?('png', 'jpg', 'jpg')
    #     attachment_exists = false
    #     status.media_attachments.each do |attachment|
    #       attachment_exists = true if attachment.remote_url == status_json['post']['url']
    #     end
    #     unless attachment_exists
    #       media_attachment = MediaAttachment.create(
    #         account: status.account,
    #         status: status,
    #         remote_url: status_json['post']['url']
    #       )
    #       media_attachment.download_file!
    #       media_attachment.download_thumbnail!
    #       media_attachment.save
    #       status.media_attachments << media_attachment
    #     end
    #   elsif !status_json['post']['url'].nil? && !status.nil?
    #     status.text = status_json['post']['name'] + "<p><a href=\"#{status_json['post']['url']}\" rel=\"nofollow noopener noreferrer\" target=\"_blank\">#{status_json['post']['url']}</a></p>"
    #     status.save!
    #   end
    #   if !status_json['post']['body'].nil? && !status.nil?
    #     status.text = status_json['post']['name'] + "<p><a href=\"#{status_json['post']['url']}\" rel=\"nofollow noopener noreferrer\" target=\"_blank\">#{status_json['post']['url']}</a></p>"
    #     status.save!
    #   end
    #   statuses.append(status) unless status.nil?
    # end
    # statuses
  end

  def set_statuses
    statuses = fetch_external_trends
    @statuses = if enabled?
                  statuses
                else
                  []
                end
  end

  def statuses_from_trends
    scope = Trends.statuses.query.allowed.in_locale(content_locale)
    scope = scope.filtered_for(current_account) if user_signed_in?
    scope
  end

  def insert_pagination_headers
    set_pagination_headers(next_path, prev_path)
  end

  def pagination_params(core_params)
    params.slice(:limit).permit(:limit).merge(core_params)
  end

  def next_path
    api_v1_trends_statuses_url pagination_params(offset: offset_param + limit_param(DEFAULT_STATUSES_LIMIT)) if records_continue?
  end

  def prev_path
    api_v1_trends_statuses_url pagination_params(offset: offset_param - limit_param(DEFAULT_STATUSES_LIMIT)) if offset_param > limit_param(DEFAULT_STATUSES_LIMIT)
  end

  def offset_param
    params[:offset].to_i
  end

  def records_continue?
    @statuses.size == limit_param(DEFAULT_STATUSES_LIMIT)
  end
end
