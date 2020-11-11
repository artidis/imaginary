# Setup build pipeline


1.  [Create the pipeline](#orgf5578a1)
2.  [Add policies the service role](#org69d4912)
3.  [Enable notifications of build failures](#org4d48417)


<a id="orgf5578a1"></a>

# Create the pipeline

-   Choose pipeline settings
    -   Pipeline name: `imaginary`
-   Source
    -   Source provider: `GitHub (Version 2)`
    -   Connect to GitHub
        -   Connection name: imaginary
        -   GitHub Apps: artidis (12370322)
    -   Repository name: artids/imaginary
    -   Branch name: master
    -   Output artifact format: Full clone
-   Build
    -   Build provider: AWS CodeBuild
    -   Region: Europe (Frankfurt)
    -   Project name
        -   Create project
            -   Project configuration
                -   Project name: imaginary
            -   Environment
                -   Environment image: Managed image
                -   Operation system: Ubuntu
                -   Runtime: Standard
                -   Image: aws/codebuild/standard:4.0
                -   Image version: Alaways use the latest image for this runtime version
                -   Environment type: Linux
                -   Privileged: *this needs to be checked for docker builds*
            -   Buildspec
                -   Build specifications: Use a buildspec file
            -   Logs
                -   CloudWatch logs
                    -   Group name: imaginary
    -   Environment variables
        -   Name: `GIT_BRANCH`
            -   Value: `#{SourceVariables.BranchName}`
            -   Type: Plaintext
        -   Name: `EA_ACCESS_TOKEN`
            -   Value: `<estudy-admin access token from https://admin.estudy.artidis.com/#/pipelines/list>`
            -   Type: Plaintext
        -   Name: `ACR_APP_ID`
            -   Value: `<Azure ACR App ID>`
            -   Type: Plaintext
        -   Name: `ACR_PASSWD`
            -   Value: `<Azure ACR Passwd>`
            -   Type: Plaintext
    -   Build type: Single build
-   Deploy
    -   Skip deploy stage


<a id="org69d4912"></a>

# Add policies the service role

<https://docs.aws.amazon.com/codepipeline/latest/userguide/troubleshooting.html#codebuild-role-connections>

Create the policy `BuildPipelineImaginaryCodeStarConnections` in IAM.

    {
        "Version": "2012-10-17",
        "Statement": [
            {
                "Effect": "Allow",
                "Action": "codestar-connections:UseConnection",
                "Resource": "arn:aws:codestar-connections:eu-central-1:421016705922:connection/28c20437-6653-4e20-bfdb-33802fe92da8"
            }
        ]
    }

You find the `Resource` ARN `arn:aws:codestar-connections:...` at: `Pipelines / imaginary / Source / (i) /
ConnectionArn`

-   Attach policies to the codebuild user
    -   Build / Build projects / imaginary / Build details / Environment / Click on `Service role`
    -   Attach policies
        -   `BuildPipelineImaginaryCodeStarConnections`
        -   `AmazonEC2ContainerRegistryPowerUser`


<a id="org4d48417"></a>

# Enable notifications of build failures

Create a new notification rule under `Pipelines / imaginary`

-   Notification rule settings
    -   Notification name: `imaginary-pipeline-failed`
-   Events that trigger notifications
    -   Pipeline execution
        -   Failed
-   Targets
    -   SNS topic
    -   code-pipeline-events

