#!/usr/bin/env ruby

require "json"
require "rugged"
require "pry"
require "benchmark"

$request = JSON.parse(STDIN.read, symbolize_names: true)

def commit_fragment(commit)
  JSON.dump({
    config: {ref: commit.oid},
    metadata: [
      {name: "author", value: enc(commit, commit.author[:name])},
      {name: "committer", value: enc(commit, commit.committer[:name])},
      {name: "message", value: enc(commit, commit.message)}
    ]
  })
end

def icon(uri)
  case uri
  when /github/
    "github-circle"
  when /gitlab/
    "gitlab"
  when /bitbucket/
    "bitbucket"
  else
    "git"
  end
end

def bench(label, &blk)
  time = Benchmark.realtime(&blk)
  $stderr.puts "#{label}: #{time}s"
end

def enc(commit, str)
  str = str.force_encoding("ISO-8859-1") unless commit.header_field("Encoding")
  str.encode("UTF-8")
end

case ARGV[0]
when "info"
  puts JSON.dump({
    "interface_version": "2.0",
    "icon": icon($request[:config][:uri]),
    "actions": {
      "check": "git-resource check",
      "get": "git-resource get",
      "put": "git-resource put"
      # delete is unsupported
    }
  })
  
when "check"
  repo =
    if File.exists?("HEAD")
      Rugged::Repository.new(".").tap do |r|
        r.fetch("origin")
      end
    else
      Rugged::Repository.clone_at(
        $request[:config][:uri],
        ".",
        checkout_branch: $request[:config][:branch],
        bare: true,
        progress: lambda { |t| $stderr.print t })
    end

  walker = Rugged::Walker.new(repo)
  walker.sorting(Rugged::SORT_TOPO|Rugged::SORT_REVERSE)
  walker.simplify_first_parent
  walker.push(repo.head.target)

  response = File.new $request[:response_path], 'w'
  total_commits = 0

  from = $request[:config][:ref]
  if from && repo.include?(from)
    commit = repo.lookup(from)
    walker.hide(commit)

    response.puts commit_fragment(commit)
    total_commits += 1
  end

  bench("walk") do
    walker.walk do |c|
      response.puts commit_fragment(c)
      total_commits += 1
    end
  end

  $stderr.puts "commits: #{total_commits}"

  response.close

when "get"
  repo =
    Rugged::Repository.clone_at(
      $request[:config][:uri],
      ".",
      checkout_branch: $request[:config][:branch])

  repo.checkout($request[:config][:ref])

  response = File.new $request[:response_path], 'w'
  response.puts commit_fragment(repo.head.target)
  response.close

when "put"
  puts "putting"
end