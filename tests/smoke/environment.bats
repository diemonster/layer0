@test "create" {
  l0 environment create env_name1
  l0 environment create --user-data common/user_data.sh --os linux --type t2.small --scale 1 env_name2
  l0 environment create env_name3
}

@test "get" {
  l0 environment get env_name1
  l0 environment get env_name2
  l0 environment get env_name*
}

@test "list" {
  l0 environment list
}

@test "link" {
  l0 environment link --bi-directional env_name1 env_name2
}
 
@test "unlink" {
  l0 environment unlink --bi-directional env_name1 env_name2
}

@test "delete" {
  l0 environment delete env_name1
  l0 environment delete env_name2
  l0 environment delete --recursive env_name3
}