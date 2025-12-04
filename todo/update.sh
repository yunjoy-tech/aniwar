make stop
svn cleanup .
svn revert -R .
svn up .
make start
exit
