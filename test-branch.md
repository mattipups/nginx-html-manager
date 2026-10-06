```
git fetch origin
git switch --track origin/feat/updates-rollback-security-v1.2.0
bash scripts/compose.sh up -d --build --wait
bash scripts/test.sh
```
