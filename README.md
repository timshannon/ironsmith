# ironsmith

Ironsmith is a simple continuous integration (build - > test -> release) tool.


## How it works

You'll setup a project which will need the following information:

1. Script to fetch from the repository
	* Most of the time this will be a git clone call, but it can be a bash script calling an FTP or whatever
	* Choose between polling for changes, or triggered builds
		* Triggered builds will be triggered off of a web hook POST call
2. Script to build the repository
3. Script to test the repository
4. Script to build the release file
5. Path to the release file
6. Script to set release name / version

Projects will be defined in a project.json file for now.  I may add a web interface later.

@dir in any of the script strings will be replaced with an absolute path to the current working directory of the specific version being worked on.
```
sh ./build.sh @dir
```


Ironsmith will take the information for the defined project above and do the following

1. Create a directory for the project
2. Change to that directory
2. Create a bolt DB file for the project to keep a log of all the builds
3. Run an initial pull of the repository using the pull script
4. Run version script
4. If pull is a new version, then Run the Build Scripts
5. If build succeeds, run the test scripts
6. If test succeeds, run the release scripts
7. Load the release file into project release folder with the release name
8. Insert the release information and the complete log into the Bolt DB file

This tool will (originally at least) have no authentication.  I plan on adding it later.


To add a new project, add a .json file to the projects/enabled folder.  Look at the template.project.json file in the projects folder for an example.
