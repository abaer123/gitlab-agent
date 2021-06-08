# Generated by the protocol buffer compiler.  DO NOT EDIT!
# source: internal/module/modserver/modserver.proto

require 'google/protobuf'

Google::Protobuf::DescriptorPool.generated_pool.build do
  add_file("internal/module/modserver/modserver.proto", :syntax => :proto3) do
    add_message "gitlab.agent.modserver.Repository" do
      optional :storage_name, :string, 2
      optional :relative_path, :string, 3
      optional :git_object_directory, :string, 4
      repeated :git_alternate_object_directories, :string, 5
      optional :gl_repository, :string, 6
      optional :gl_project_path, :string, 8
    end
    add_message "gitlab.agent.modserver.GitalyAddress" do
      optional :address, :string, 1
      optional :token, :string, 2
    end
  end
end

module Gitlab
  module Agent
    module Modserver
      Repository = ::Google::Protobuf::DescriptorPool.generated_pool.lookup("gitlab.agent.modserver.Repository").msgclass
      GitalyAddress = ::Google::Protobuf::DescriptorPool.generated_pool.lookup("gitlab.agent.modserver.GitalyAddress").msgclass
    end
  end
end