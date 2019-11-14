# Downloads

To download `Edgectl` to configure the Ambassador Edge Stack interface, download the file directly from one of these links:

* MacOS: https://metriton.datawire.io/downloads/darwin/edgectl

* Linux: https://metriton.datawire.io/downloads/linux/edgectl

## MacOS Permissions Error

When you try to open the `Edgectl` file from your Ambassador Edge Stack UI, it may indicate that `Edgectl` is not safe to download and run. The command from the initial Ambassador Edge Stack UI pages (`edgectl login <your IP>`) also may not be recognized as a command.

To successfully install and run `Edgectl`:

1. From the Ambassador Edge Stack UI, click on the Darwin link to download the file for MacOS, or click the MacOS link above.
2. Find the file where it downloaded and move it to the repository where your Ambassador Edge Stack files are located.
3. In your command line interface, navigate to your Ambassador Edge Stack files.
    * Use `ls` to list what items are available to you. Use `cd <folder>` to navigate into a folder, or just `cd` to move up towards your home directory. For example, `cd documents/github/my repo` moves you into the "my repo" folder.
4. The file *is* executable, but MacOS will not allow you to run it. You will see "Permission denied" when you run the following command: `./edgectl login <your IP>`
5. To make `Edgectl` an executable that you can run, use following command: `chmod a+x edgectl` and then `./edgectl login <your IP>`
6. A warning popup will appear indicating that the file is not from a verified developer. Click the "OK" button.
7. To allow permission, go to MacOS System Preferences > Security & Privacy > "Allow Apps downloaded from" > App Store. You may need to unlock with your password to change this setting.
8. You will see a message indicating that `edgectl` was recently blocked. Click the "Allow Anyway" button that is present.
9. Return to the command line tool and run `./edgectl login <your IP>`
10. When the Warning dialog appears, click the "Open" button.

You should see a success message for `Edgectl` which will open your Ambassador Edge Stack UI.
