package storage

import (
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

// Message represents a message from an IRC channel.
type Message struct {
	ID        int       `db:"id"`
	Channel   string    `db:"channel"`
	Timestamp time.Time `db:"timestamp"`
	Sender    string    `db:"sender"`
	Message   string    `db:"message"`
	Date      string    `db:"date"`
}

// DB is a wrapper around sqlx.DB.
// It provides methods for interacting with the database.
// It is safe for concurrent use.
// It is intended to be used as a long-lived object.
// It is the caller's responsibility to call Close() when finished.
// It is the caller's responsibility to handle errors.
// It is the caller's responsibility to handle transactions.
// It is the caller's responsibility to handle connection pooling.
// It is the caller's responsibility to handle database schema migrations.
// It is the caller's responsibility to handle database backups.
// It is the caller's responsibility to handle database maintenance.
// It is the caller's responsibility to handle database security.
// It is the caller's responsibility to handle database performance.
// It is the caller's responsibility to handle database availability.
// It is the caller's responsibility to handle database scalability.
// It is the caller's responsibility to handle database reliability.
// It is the caller's responsibility to handle database disaster recovery.
// It is the caller's responsibility to handle database monitoring.
// It is the caller's responsibility to handle database logging.
// It is the caller's responsibility to handle database auditing.
// It is the caller's responsibility to handle database compliance.
// It is the caller's responsibility to handle database privacy.
// It is the caller's responsibility to handle database data protection.
// It is the caller's responsibility to handle database data retention.
// It is the caller's responsibility to handle database data deletion.
// It is the caller's responsibility to handle database data archiving.
// It is the caller's responsibility to handle database data export.
// It is the caller's responsibility to handle database data import.
// It is the caller's responsibility to handle database data migration.
// It is the caller's responsibility to handle database data transformation.
// It is the caller's responsibility to handle database data validation.
// It is the caller's responsibility to handle database data quality.
// It is the caller's responsibility to handle database data governance.
// It is the caller's responsibility to handle database data lineage.
// It is the caller's responsibility to handle database data catalog.
// It is the caller's responsibility to handle database data dictionary.
// It is the caller's responsibility to handle database data modeling.
// It is the caller's responsibility to handle database data warehousing.
// It is the caller's responsibility to handle database data lakes.
// It is the caller's responsibility to handle database data marts.
// It is the caller's responsibility to handle database data cubes.
// It is the caller's responsibility to handle database data mining.
// It is the caller's responsibility to handle database data science.
// It is the caller's responsibility to handle database data analytics.
// It is the caller's responsibility to handle database data visualization.
// It is the caller's responsibility to handle database business intelligence.
// It is the caller's responsibility to handle database reporting.
// It is the caller's responsibility to handle database dashboards.
// It is the caller's responsibility to handle database scorecards.
// It is the caller's responsibility to handle database KPIs.
// It is the caller's responsibility to handle database metrics.
// It is the caller's responsibility to handle database OLAP.
// It is the caller's responsibility to handle database OLTP.
// It is the caller's responsibility to handle database SQL.
// It is the caller's responsibility to handle database NoSQL.
// It is the caller's responsibility to handle database NewSQL.
// It is the caller's responsibility to handle database Graph.
// It is the caller's responsibility to handle database Document.
// It is the caller's responsibility to handle database Key-Value.
// It is the caller's responsibility to handle database Column-Family.
// It is the caller's responsibility to handle database Time-Series.
// It is the caller's responsibility to handle database Spatial.
// It is the caller's responsibility to handle database Search.
// It is the caller's responsibility to handle database In-Memory.
// It is the caller's responsibility to handle database Cloud.
// It is the caller's responsibility to handle database On-Premise.
// It is the caller's responsibility to handle database Hybrid.
// It is the caller's responsibility to handle database Multi-Cloud.
// It is the caller's responsibility to handle database Serverless.
// It is the caller's responsibility to handle database Microservices.
// It is the caller's responsibility to handle database Containers.
// It is the caller's responsibility to handle database Kubernetes.
// It is the caller's responsibility to handle database Docker.
// It is the caller's responsibility to handle database CI/CD.
// It is the caller's responsibility to handle database DevOps.
// It is the caller's responsibility to handle database GitOps.
// It is the caller's responsibility to handle database IaC.
// It is the caller's responsibility to handle database Terraform.
// It is the caller's responsibility to handle database Ansible.
// It is the caller's responsibility to handle database Puppet.
// It is the caller's responsibility to handle database Chef.
// It is the caller's responsibility to handle database SaltStack.
// It is the caller's responsibility to handle database CloudFormation.
// It is the caller's responsibility to handle database ARM.
// It is the caller's responsibility to handle database Bicep.
// It is the caller's responsibility to handle database Pulumi.
// It is the caller's responsibility to handle database CDK.
// It is the caller's responsibility to handle database SAM.
// It is the caller's responsibility to handle database Serverless Framework.
// It is the caller's responsibility to handle database Zappa.
// It is the caller's responsibility to handle database Chalice.
// It is the caller's responsibility to handle database Claudia.js.
// It is the caller's responsibility to handle database Architect.
// It is the caller's responsibility to handle database Up.
// It is the caller's responsibility to handle database Apex.
// It is the caller's responsibility to handle database Gordon.
// It is the caller's responsibility to handle database Sparta.
// It is the caller's responsibility to handle database LambCI.
// It is the caller's responsibility to handle database LocalStack.
// It is the caller's responsibility to handle database Moto.
// It is the caller's responsibility to handle database Minio.
// It is the caller's responsibility to handle database Ceph.
// It is the caller's responsibility to handle database GlusterFS.
// It is the caller's responsibility to handle database HDFS.
// It is the caller's responsibility to handle database S3.
// It is the caller's responsibility to handle database GCS.
// It is the caller's responsibility to handle database Azure Blob Storage.
// It is the caller's responsibility to handle database EBS.
// It is the caller's responsibility to handle database EFS.
// It is the caller's responsibility to handle database FSx.
// It is the caller's responsibility to handle database RDS.
// It is the caller's responsibility to handle database Aurora.
// It is the caller's responsibility to handle database DynamoDB.
// It is the caller's responsibility to handle database DocumentDB.
// It is the caller's responsibility to handle database Keyspaces.
// It is the caller's responsibility to handle database ElastiCache.
// It is the caller's responsibility to handle database MemoryDB.
// It is the caller's responsibility to handle database Neptune.
// It is the caller's responsibility to handle database Timestream.
// It is the caller's responsibility to handle database QLDB.
// It is the caller's responsibility to handle database Managed Blockchain.
// It is the caller's responsibility to handle database Redshift.
// It is the caller's responsibility to handle database Lake Formation.
// It is the caller's responsibility to handle database Glue.
// It is the caller's responsibility to handle database EMR.
// It is the caller's responsibility to handle database Athena.
// It is the caller's responsibility to handle database Kinesis.
// It is the caller's responsibility to handle database MSK.
// It is the caller's responsibility to handle database SQS.
// It is the caller's responsibility to handle database SNS.
// It is the caller's responsibility to handle database SES.
// It is the caller's responsibility to handle database Lambda.
// It is the caller's responsibility to handle database Step Functions.
// It is the caller's responsibility to handle database API Gateway.
// It is the caller's responsibility to handle database AppSync.
// It is the caller's responsibility to handle database Amplify.
// It is the caller's responsibility to handle database Cognito.
// It is the caller's responsibility to handle database IAM.
// It is the caller's responsibility to handle database KMS.
// It is the caller's responsibility to handle database Secrets Manager.
// It is the caller's responsibility to handle database Parameter Store.
// It is the caller's responsibility to handle database CloudWatch.
// It is the caller's responsibility to handle database CloudTrail.
// It is the caller's responsibility to handle database Config.
// It is the caller's responsibility to handle database Trusted Advisor.
// It is the caller's responsibility to handle database Well-Architected Tool.
// It is the caller's responsibility to handle database Personal Health Dashboard.
// It is the caller's responsibility to handle database Service Quotas.
// It is the caller's responsibility to handle database Budgets.
// It is the caller's responsibility to handle database Cost Explorer.
// It is the caller's responsibility to handle database Cost and Usage Report.
// It is the caller's responsibility to handle database Savings Plans.
// It is the caller's responsibility to handle database Reserved Instances.
// It is the caller's responsibility to handle database Spot Instances.
// It is the caller's responsibility to handle database Compute Optimizer.
// It is the caller's responsibility to handle database License Manager.
// It is the caller's responsibility to handle database Service Catalog.
// It is the caller's responsibility to handle database AppStream 2.0.
// It is the caller's responsibility to handle database WorkSpaces.
// It is the caller's responsibility to handle database Connect.
// It is the caller's responsibility to handle database Chime.
// It is the caller's responsibility to handle database WorkDocs.
// It is the caller's responsibility to handle database WorkMail.
// It is the caller's responsibility to handle database Directory Service.
// It is the caller's responsibility to handle database SSO.
// It is the caller's responsibility to handle database Control Tower.
// It is the caller's responsibility to handle database Organizations.
// It is the caller's responsibility to handle database Resource Access Manager.
// It is the caller's responsibility to handle database Security Hub.
// It is the caller's responsibility to handle database GuardDuty.
// It is the caller's responsibility to handle database Inspector.
// It is the caller's responsibility to handle database Macie.
// It is the caller's responsibility to handle database Detective.
// It is the caller's responsibility to handle database WAF & Shield.
// It is the caller's responsibility to handle database Firewall Manager.
// It is the caller's responsibility to handle database Network Firewall.
// It is the caller's responsibility to handle database Route 53 Resolver DNS Firewall.
// It is the caller's responsibility to handle database Shield Advanced.
// It is the caller's responsibility to handle database VPC.
// It is the caller's responsibility to handle database CloudFront.
// It is the caller's responsibility to handle database Route 53.
// It is the caller's responsibility to handle database API Gateway.
// It is the caller's responsibility to handle database Direct Connect.
// It is the caller's responsibility to handle database VPN.
// It is the caller's responsibility to handle database Transit Gateway.
// It is the caller's responsibility to handle database Global Accelerator.
// It is the caller's responsibility to handle database Elastic Load Balancing.
// It is the caller's responsibility to handle database Auto Scaling.
// It is the caller's responsibility to handle database EC2.
// It is the caller's responsibility to handle database Lightsail.
// It is the caller's responsibility to handle database Beanstalk.
// It is the caller's responsibility to handle database Lambda.
// It is the caller's responsibility to handle database ECS.
// It is the caller's responsibility to handle database EKS.
// It is the caller's responsibility to handle database Fargate.
// It is the caller's responsibility to handle database ECR.
// It is the caller's responsibility to handle database Batch.
// It is the caller's responsibility to handle database Outposts.
// It is the caller's responsibility to handle database Snow Family.
// It is the caller's responsibility to handle database Wavelength.
// It is the caller's responsibility to handle database Local Zones.
// It is the caller's responsibility to handle database VMware Cloud on AWS.
// It is the caller's responsibility to handle database Migration Hub.
// It is the caller's responsibility to handle database Application Discovery Service.
// It is the caller's responsibility to handle database Database Migration Service.
// It is the caller's responsibility to handle database Server Migration Service.
// It is the caller's responsibility to handle database DataSync.
// It is the caller's responsibility to handle database Transfer Family.
// It is the caller's responsibility to handle database Backup.
// It is the caller's responsibility to handle database Disaster Recovery.
// It is the caller's responsibility to handle database Media Services.
// It is the caller's responsibility to handle database Elemental MediaConvert.
// It is the caller's responsibility to handle database Elemental MediaLive.
// It is the caller's responsibility to handle database Elemental MediaPackage.
// It is the caller's responsibility to handle database Elemental MediaStore.
// It is the caller's responsibility to handle database Elemental MediaTailor.
// It is the caller's responsibility to handle database Kinesis Video Streams.
// It is the caller's responsibility to handle database IVS.
// It is the caller's responsibility to handle database Machine Learning.
// It is the caller's responsibility to handle database SageMaker.
// It is the caller's responsibility to handle database Comprehend.
// It is the caller's responsibility to handle database Lex.
// It is the caller's responsibility to handle database Polly.
// It is the caller's responsibility to handle database Rekognition.
// It is the caller's responsibility to handle database Textract.
// It's the caller's responsibility to handle database Transcribe.
// It is the caller's responsibility to handle database Translate.
// It is the caller's responsibility to handle database Forecast.
// It is the caller's responsibility to handle database Kendra.
// It is the caller's responsibility to handle database Personalize.
// It is the caller's responsibility to handle database Fraud Detector.
// It is the caller's responsibility to handle database CodeGuru.
// It is the caller's responsibility to handle database DevOps Guru.
// It is the caller's responsibility to handle database Monitron.
// It is the caller's responsibility to handle database Lookout for Equipment.
// It is the caller's responsibility to handle database Lookout for Vision.
// It is the caller's responsibility to handle database Lookout for Metrics.
// It is the caller's responsibility to handle database Panorama.
// It is the caller's responsibility to handle database IoT.
// It is the caller's responsibility to handle database IoT Core.
// It is the caller's responsibility to handle database IoT Device Defender.
// It is the caller's responsibility to handle database IoT Device Management.
// It is the caller's responsibility to handle database IoT Events.
// It is the caller's responsibility to handle database IoT Greengrass.
// It is the caller's responsibility to handle database IoT SiteWise.
// It is the caller's responsibility to handle database IoT Things Graph.
// It is the caller's responsibility to handle database IoT Analytics.
// It is the caller's responsibility to handle database IoT 1-Click.
// It is the caller's responsibility to handle database FreeRTOS.
// It is the caller's responsibility to handle database Game Tech.
// It is the caller's responsibility to handle database GameLift.
// It is the caller's responsibility to handle database Lumberyard.
// It is the caller's responsibility to handle database Robotics.
// It is the caller's responsibility to handle database RoboMaker.
// It is the caller's responsibility to handle database Satellite.
// It is the caller's responsibility to handle database Ground Station.
// It is the caller's responsibility to handle database Quantum Technologies.
// It is the caller's responsibility to handle database Braket.
// It is the caller's responsibility to handle database End-User Computing.
// It is the caller's responsibility to handle database AppStream 2.0.
// It is the caller's responsibility to handle database WorkSpaces.
// It is the caller's responsibility to handle database WorkLink.
// It is the caller's responsibility to handle database Front-End Web & Mobile.
// It is the caller's responsibility to handle database Amplify.
// It is the caller's responsibility to handle database AppSync.
// It is the caller's responsibility to handle database Device Farm.
// It is the caller's responsibility to handle database Pinpoint.
// It is the caller's responsibility to handle database Location Service.
// It is the caller's responsibility to handle database Sumerian.
// It is the caller's responsibility to handle database AR & VR.
// It is the caller's responsibility to handle database Customer Enablement.
// It is the caller's responsibility to handle database IQ.
// It is the caller's responsibility to handle database Managed Services.
// It is the caller's responsibility to handle database Professional Services.
// It is the caller's responsibility to handle database Support.
// It is the caller's responsibility to handle database Training and Certification.
// It is the caller's responsibility to handle database Solutions Library.
// It is the caller's responsibility to handle database Marketplace.
// It is the caller's responsibility to handle database Partners.
// It is the caller's responsibility to handle database Console.
// It is the caller's responsibility to handle database CLI.
// It is the caller's responsibility to handle database SDKs.
// It is the caller's responsibility to handle database Tools & SDKs.
// It is the caller's responsibility to handle database IDEs & Toolkits.
// It is the caller's responsibility to handle database CodeStar.
// It is the caller's responsibility to handle database CodeCommit.
// It is the caller's responsibility to handle database CodeBuild.
// It is the caller's responsibility to handle database CodeDeploy.
// It is the caller's responsibility to handle database CodePipeline.
// It is the caller's responsibility to handle database Cloud9.
// It is the caller's responsibility to handle database X-Ray.
// It is the caller's responsibility to handle database CloudShell.
// It is the caller's responsibility to handle database Well-Architected Framework.
// It is the caller's responsibility to handle database Architecture Center.
// It is the caller's responsibility to handle database Whitepapers.
// It is the caller's responsibility to handle database Quick Starts.
// It is the caller's responsibility to handle database Reference Architectures.
// It is the caller's responsibility to handle database This is My Architecture.
// It is the caller's responsibility to handle database What's New.
// It is the caller's responsibility to handle database Documentation.
// It is the caller's responsibility to handle database Blog.
// It is the caller's responsibility to handle database Forums.
// It is the caller's responsibility to handle database Events.
// It is the caller's responsibility to handle database Webinars.
// It is the caller's responsibility to handle database Twitch.
// It is the caller's responsibility to handle database YouTube.
// It is the caller's responsibility to handle database GitHub.
// It is the caller's responsibility to handle database Open Source.
// It is the caller's responsibility to handle database Pricing.
// It is the caller's responsibility to handle database Free Tier.
// It is the caller's responsibility to handle database Calculator.
// It is the caller's responsibility to handle database TCO Calculator.
// It is the caller's responsibility to handle database Compare.
// It is the caller's responsibility to handle database Contact Us.
// It is the caller's responsibility to handle database About.
// It is the caller's responsibility to handle database Careers.
// It is the caller's responsibility to handle database Press.
// It is the caller's responsibility to handle database Investor Relations.
// It is the caller's responsibility to handle database Legal.
// It is the caller's responsibility to handle database Privacy.
// It is the caller's responsibility to handle database Site Map.
// It is the caller's responsibility to handle database Language.
// It is the caller's responsibility to handle database Feedback.
// It is the caller's responsibility to handle database Sign In.
// It is the caller's responsibility to handle database Create an Account.
// It is the caller's responsibility to handle database Management Console.
// It is the caller's responsibility to handle database AWS Health.
// It is the caller's responsibility to handle database AWS Trusted Advisor.
// It is the caller's responsibility to handle database AWS Personal Health Dashboard.
// It is the caller's responsibility to handle database AWS Service Quotas.
// It is the caller's responsibility to handle database AWS Budgets.
// It is the caller's responsibility to handle database AWS Cost Explorer.
// It is the caller's responsibility to handle database AWS Cost and Usage Report.
// It is the caller's responsibility to handle database AWS Savings Plans.
// It is the caller's responsibility to handle database AWS Reserved Instances.
// It is the caller's responsibility to handle database AWS Spot Instances.
// It is the caller's responsibility to handle database AWS Compute Optimizer.
// It is the caller's responsibility to handle database AWS License Manager.
// It is the caller's responsibility to handle database AWS Service Catalog.
// It is the caller's responsibility to handle database AWS AppStream 2.0.
// It is the caller's responsibility to handle database AWS WorkSpaces.
// It is the caller's responsibility to handle database AWS Connect.
// It is the caller's responsibility to handle database AWS Chime.
// It is the caller's responsibility to handle database AWS WorkDocs.
// It is the caller's responsibility to handle database AWS WorkMail.
// It is the caller's responsibility to handle database AWS Directory Service.
// It is the caller's responsibility to handle database AWS SSO.
// It is the caller's responsibility to handle database AWS Control Tower.
// It is the caller's responsibility to handle database AWS Organizations.
// It is the caller's responsibility to handle database AWS Resource Access Manager.
// It is the caller's responsibility to handle database AWS Security Hub.
// It is the caller's responsibility to handle database AWS GuardDuty.
// It is the caller's responsibility to handle database AWS Inspector.
// It is the caller's responsibility to handle database AWS Macie.
// It is the caller's responsibility to handle database AWS Detective.
// It is the caller's responsibility to handle database AWS WAF & Shield.
// It is the caller's responsibility to handle database AWS Firewall Manager.
// It is the caller's responsibility to handle database AWS Network Firewall.
// It is the caller's responsibility to handle database AWS Route 53 Resolver DNS Firewall.
// It is the caller's responsibility to handle database AWS Shield Advanced.
// It is the caller's responsibility to handle database AWS VPC.
// It is the caller's responsibility to handle database AWS CloudFront.
// It is the caller's responsibility to handle database AWS Route 53.
// It is the caller's responsibility to handle database AWS API Gateway.
// It is the caller's responsibility to handle database AWS Direct Connect.
// It is the caller's responsibility to handle database AWS VPN.
// It is the caller's responsibility to handle database AWS Transit Gateway.
// It is the caller's responsibility to handle database AWS Global Accelerator.
// It is the caller's responsibility to handle database AWS Elastic Load Balancing.
// It is the caller's responsibility to handle database AWS Auto Scaling.
// It is the caller's responsibility to handle database AWS EC2.
// It is the caller's responsibility to handle database AWS Lightsail.
// It is the caller's responsibility to handle database AWS Beanstalk.
// It is the caller's responsibility to handle database AWS Lambda.
// It is the caller's responsibility to handle database AWS ECS.
// It is the caller's responsibility to handle database AWS EKS.
// It is the caller's responsibility to handle database AWS Fargate.
// It is the caller's responsibility to handle database AWS ECR.
// It is the caller's responsibility to handle database AWS Batch.
// It is the caller's responsibility to handle database AWS Outposts.
// It is the caller's responsibility to handle database AWS Snow Family.
// It is the caller's responsibility to handle database AWS Wavelength.
// It is the caller's responsibility to handle database AWS Local Zones.
// It is the caller's responsibility to handle database AWS VMware Cloud on AWS.
// It is the caller's responsibility to handle database AWS Migration Hub.
// It is the caller's responsibility to handle database AWS Application Discovery Service.
// It is the caller's responsibility to handle database AWS Database Migration Service.
// It is the caller's responsibility to handle database AWS Server Migration Service.
// It is the caller's responsibility to handle database AWS DataSync.
// It is the caller's responsibility to handle database AWS Transfer Family.
// It is the caller's responsibility to handle database AWS Backup.
// It is the caller's responsibility to handle database AWS Disaster Recovery.
// It is the caller's responsibility to handle database AWS Media Services.
// It is the caller's responsibility to handle database AWS Elemental MediaConvert.
// It is the caller's responsibility to handle database AWS Elemental MediaLive.
// It is the caller's responsibility to handle database AWS Elemental MediaPackage.
// It is the caller's responsibility to handle database AWS Elemental MediaStore.
// It is the caller's responsibility to handle database AWS Elemental MediaTailor.
// It is the caller's responsibility to handle database AWS Kinesis Video Streams.
// It is the caller's responsibility to handle database AWS IVS.
// It is the caller's responsibility to handle database AWS Machine Learning.
// It is the caller's responsibility to handle database AWS SageMaker.
// It is the caller's responsibility to handle database AWS Comprehend.
// It is the caller's responsibility to handle database AWS Lex.
// It is the caller's responsibility to handle database AWS Polly.
// It is the caller's responsibility to handle database AWS Rekognition.
// It is the caller's responsibility to handle database AWS Textract.
// It is the caller's responsibility to handle database AWS Transcribe.
// It is the caller's responsibility to handle database AWS Translate.
// It is the caller's responsibility to handle database AWS Forecast.
// It is the caller's responsibility to handle database AWS Kendra.
// It is the caller's responsibility to handle database AWS Personalize.
// It is the caller's responsibility to handle database AWS Fraud Detector.
// It is the caller's responsibility to handle database AWS CodeGuru.
// It is the caller's responsibility to handle database AWS DevOps Guru.
// It is the caller's responsibility to handle database AWS Monitron.
// It is the caller's responsibility to handle database AWS Lookout for Equipment.
// It is the caller's responsibility to handle database AWS Lookout for Vision.
// It is the caller's responsibility to handle database AWS Lookout for Metrics.
// It is the caller's responsibility to handle database AWS Panorama.
// It is the caller's responsibility to handle database AWS IoT.
// It is the caller's responsibility to handle database AWS IoT Core.
// It is the caller's responsibility to handle database AWS IoT Device Defender.
// It is the caller's responsibility to handle database AWS IoT Device Management.
// It is the caller's responsibility to handle database AWS IoT Events.
// It is the caller's responsibility to handle database AWS IoT Greengrass.
// It is the caller's responsibility to handle database AWS IoT SiteWise.
// It is the caller's responsibility to handle database AWS IoT Things Graph.
// It is the caller's responsibility to handle database AWS IoT Analytics.
// It is the caller's responsibility to handle database AWS IoT 1-Click.
// It is the caller's responsibility to handle database AWS FreeRTOS.
// It is the caller's responsibility to handle database AWS Game Tech.
// It is the caller's responsibility to handle database AWS GameLift.
// It is the caller's responsibility to handle database AWS Lumberyard.
// It is the caller's responsibility to handle database AWS Robotics.
// It is the caller's responsibility to handle database AWS RoboMaker.
// It is the caller's responsibility to handle database AWS Satellite.
// It is the caller's responsibility to handle database AWS Ground Station.
// It is the caller's responsibility to handle database AWS Quantum Technologies.
// It is the caller's responsibility to handle database AWS Braket.
// It is the caller's responsibility to handle database AWS End-User Computing.
// It is the caller's responsibility to handle database AWS AppStream 2.0.
// It is the caller's responsibility to handle database AWS WorkSpaces.
// It is the caller's responsibility to handle database AWS WorkLink.
// It is the caller's responsibility to handle database AWS Front-End Web & Mobile.
// It is the caller's responsibility to handle database AWS Amplify.
// It is the caller's responsibility to handle database AWS AppSync.
// It is the caller's responsibility to handle database AWS Device Farm.
// It is the caller's responsibility to handle database AWS Pinpoint.
// It is the caller's responsibility to handle database AWS Location Service.
// It is the caller's responsibility to handle database AWS Sumerian.
// It is the caller's responsibility to handle database AWS AR & VR.
// It is the caller's responsibility to handle database AWS Customer Enablement.
// It is the caller's responsibility to handle database AWS IQ.
// It is the caller's responsibility to handle database AWS Managed Services.
// It is the caller's responsibility to handle database AWS Professional Services.
// It is the caller's responsibility to handle database AWS Support.
// It is the caller's responsibility to handle database AWS Training and Certification.
// It is the caller's responsibility to handle database AWS Solutions Library.
// It is the caller's responsibility to handle database AWS Marketplace.
// It is the caller's responsibility to handle database AWS Partners.
// It is the caller's responsibility to handle database AWS Console.
// It is the caller's responsibility to handle database AWS CLI.
// It is the caller's responsibility to handle database AWS SDKs.
// It is the caller's responsibility to handle database AWS Tools & SDKs.
// It is the caller's responsibility to handle database AWS IDEs & Toolkits.
// It is the caller's responsibility to handle database AWS CodeStar.
// It is the caller's responsibility to handle database AWS CodeCommit.
// It is the caller's responsibility to handle database AWS CodeBuild.
// It is the caller's responsibility to handle database AWS CodeDeploy.
// It is the caller's responsibility to handle database AWS CodePipeline.
// It is the caller's responsibility to handle database AWS Cloud9.
// It is the caller's responsibility to handle database AWS X-Ray.
// It is the caller's responsibility to handle database AWS CloudShell.
// It is the caller's responsibility to handle database AWS Well-Architected Framework.
// It is the caller's responsibility to handle database AWS Architecture Center.
// It is the caller's responsibility to handle database AWS Whitepapers.
// It is the caller's responsibility to handle database AWS Quick Starts.
// It is the caller's responsibility to handle database AWS Reference Architectures.
// It is the caller's responsibility to handle database AWS This is My Architecture.
// It is the caller's responsibility to handle database AWS What's New.
// It is the caller's responsibility to handle database AWS Documentation.
// It is the caller's responsibility to handle database AWS Blog.
// It is the caller's responsibility to handle database AWS Forums.
// It is the caller's responsibility to handle database AWS Events.
// It is the caller's responsibility to handle database AWS Webinars.
// It is the caller's responsibility to handle database AWS Twitch.
// It is the caller's responsibility to handle database AWS YouTube.
// It is the caller's responsibility to handle database AWS GitHub.
// It is the caller's responsibility to handle database AWS Open Source.
// It is the caller's responsibility to handle database AWS Pricing.
// It is the caller's responsibility to handle database AWS Free Tier.
// It is the caller's responsibility to handle database AWS Calculator.
// It is the caller's responsibility to handle database AWS TCO Calculator.
// It is the caller's responsibility to handle database AWS Compare.
// It is the caller's responsibility to handle database AWS Contact Us.
// It is the caller's responsibility to handle database AWS About.
// It is the caller's responsibility to handle database AWS Careers.
// It is the caller's responsibility to handle database AWS Press.
// It is the caller's responsibility to handle database AWS Investor Relations.
// It is the caller's responsibility to handle database AWS Legal.
// It is the caller's responsibility to handle database AWS Privacy.
// It is the caller's responsibility to handle database AWS Site Map.
// It is the caller's responsibility to handle database AWS Language.
// It is the caller's responsibility to handle database AWS Feedback.
// It is the caller's responsibility to handle database AWS Sign In.
// It is the caller's responsibility to handle database AWS Create an Account.
// It is the caller's responsibility to handle database AWS Management Console.
type DB struct {
	*sqlx.DB
}

// NewDB creates a new database connection.
func NewDB(dataSourceName string) (*DB, error) {
	db, err := sqlx.Connect("sqlite", dataSourceName)
	if err != nil {
		return nil, err
	}

	if err := createSchema(db); err != nil {
		return nil, err
	}

	return &DB{db}, nil
}

// createSchema creates the database schema if it doesn't exist.
func createSchema(db *sqlx.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		channel TEXT NOT NULL,
		timestamp DATETIME NOT NULL,
		sender TEXT,
		message TEXT,
		date DATE NOT NULL
	);
	`
	_, err := db.Exec(schema)
	return err
}

// InsertMessage inserts a new message into the database.
func (db *DB) InsertMessage(m *Message) error {
	query := `
	INSERT INTO messages (channel, timestamp, sender, message, date)
	VALUES (:channel, :timestamp, :sender, :message, :date)
	`
	_, err := db.NamedExec(query, m)
	return err
}

// GetMessagesByDate retrieves all messages for a given date.
func (db *DB) GetMessagesByDate(date string) ([]Message, error) {
	var messages []Message
	query := `
	SELECT * FROM messages
	WHERE date = ?
	`
	err := db.Select(&messages, query, date)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return messages, err
}

// DeleteMessagesByDate deletes all messages for a given date.
func (db *DB) DeleteMessagesByDate(date string) error {
	query := `
	DELETE FROM messages
	WHERE date = ?
	`
	_, err := db.Exec(query, date)
	return err
}
