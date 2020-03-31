provider "aws" {
  region = "us-east-1"
}

resource "aws_iam_access_key" "aes-backend-dns" {
  user = "${aws_iam_user.aes-backend-dns.name}"
  pgp_key = "keybase:alexgervais_dw"
}

resource "aws_iam_user" "aes-backend-dns" {
  name = "aes-backend-dns"
}

resource "aws_iam_policy" "aes-backend-dns" {
  name = "aes-backend-dns"
  description = "AES Backend DNS policy for easy DNS registration and crash-report uploads to S3"
  policy = <<EOF
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "s3:PutObject",
                "route53:ChangeResourceRecordSets"
            ],
            "Resource": [
                "arn:aws:s3:::datawire-crash-reports/*",
                "arn:aws:route53:::hostedzone/Z80AVJZQXUM45"
            ]
        }
    ]
}
EOF
}

resource "aws_iam_user_policy_attachment" "aes-backend-dns" {
  user = "${aws_iam_user.aes-backend-dns.name}"
  policy_arn = "${aws_iam_policy.aes-backend-dns.arn}"
}

output "secret" {
  value = "${aws_iam_access_key.aes-backend-dns.encrypted_secret}"
}
