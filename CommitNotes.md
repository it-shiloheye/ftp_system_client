# Commit Notes

### 14th June 2024, 01:45 AM GMT+3
```sh
1. Removing logging to lib package
2. To standardise logging both serverside 
    and client side
# Please enter the commit message for your changes. Lines starting
# with '#' will be ignored, and an empty message aborts the commit.
#
# On branch main
# Your branch is up to date with 'origin/main'.
#
# Changes to be committed:
#	modified:   go.mod
#	modified:   init_client/config.go
#	modified:   main.go
#	deleted:    main_thread/logging/logging_struct.go
#
```

### 13th June 2024, 10:19 AM GMT+3
```sh
1. Successfully uploads files as file_tree_json 
    (want to switch to bytestream)
2. tracks which files have already been uploaded
    (to not resend)
3. Able to hash successfully
    (going for "single threaded" design)
# Please enter the commit message for your changes. Lines starting
# with '#' will be ignored, and an empty message aborts the commit.
#
# On branch main
# Your branch is up to date with 'origin/main'.
#
# Changes to be committed:
#	modified:   main_thread/dir_handler/file_tree_json.go
#	modified:   main_thread/dir_handler/read_files_in_directory.go
#	modified:   main_thread/logging/logging_struct.go
#	modified:   main_thread/main_thread.go
#	modified:   main_thread/network_client/network_client.go
#
```


### 08th June 2024, 19:33 PM GMT+3
    1. Set up client git repo
    2. want to use client structs in server
    3. Able to:
        a. list files in ClientConfig.DataDir
        b. upload file metadata to server
        c. upload file to server