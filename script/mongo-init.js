use game
db.player.drop()
db.createCollection("player")
use account
db.user.drop()
db.createCollection("user")
use mail
db.mail.drop()
db.createCollection("mail")
use gmt
db.gmt.drop()
db.createCollection("gmt")

sh.enableSharding("game")
sh.enableSharding("account")
sh.enableSharding("mail")
sh.enableSharding("gmt")

sh.shardCollection("game.player", {"_id": "hashed" })
sh.shardCollection("account.user", {"_id": "hashed" })
sh.shardCollection("mail.mail", {"_id": "hashed" })
sh.shardCollection("gmt.gmt", {"_id": "hashed" })

use game
db.player.getShardDistribution()

use account
db.user.getShardDistribution()

use mail
db.mail.getShardDistribution()

use gmt
db.gmt.getShardDistribution()
