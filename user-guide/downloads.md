# Downloads

To download `Edgectl` to configure the Ambassador Edge Stack interface, download the file directly from one of these links: 

* MacOS: https://metriton.datawire.io/downloads/darwin/edgectl

* Linux: https://metriton.datawire.io/downloads/linux/edgectl

## MacOS Permissions Error

When you try to open the `Edgectl` file from your Ambassador Edge Stack UI, it may indicate that `Edgectl` is not safe to download and run. The command from the initial Ambassador Edge Stack UI pages (`edgectl login <your IP>`) also may not be recognized as a command. 

To successfully install and run `Edgectl`:

1. From the Ambassador Edge Stack UI, click on the Darwin link to download the file for MacOS, or click the MacOS link above.
2. Find the file and move it to the repository where your Ambassador Edge Stack files are located.
3. To run `Edgectl` you must first make it an executable. Do so with the following command: `chmod a+x edgectl`
4. The file is now executable, but MacOS will not allow you to run it. You will see "Permission denied" when you run the following command: `./edgectl login <your IP>` 
5. To allow permission, go to  MacOS System Preferences > Security & Privacy > "Allow Apps downloaded from" > App Store. You may need to unlock with your password to change this setting.
6. Click the "Allow Anyway" button that is present.
7. Return to the command line tool and run `./edgectl login <your IP>`
8. When the Warning dialog appears, click the "Open" button. 

You should see a success message for `Edgectl` which will open your Ambassador Edge Stack UI. 
