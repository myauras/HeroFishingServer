module crontasker

go 1.21.0

replace herofishingGoModule => ../herofishingGoModule // for local

// replace herofishingGoModule => /home/herofishingGoModule // for docker

require github.com/robfig/cron/v3 v3.0.1 // indirect
