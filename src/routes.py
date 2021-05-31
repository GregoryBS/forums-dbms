from .views import *

def setup_routes(app):
    app.router.add_get('/', index)
    app.router.add_post('/api/user/{nick}/create', signup, name = 'signup')
    app.router.add_get('/api/user/{nick}/profile', get_profile, name = 'personal')
    app.router.add_post('/api/user/{nick}/profile', update_profile, name = 'personal_edit')
    app.router.add_post('/api/forum/create', create_forum, name = 'new_forum')
    app.router.add_get('/api/forum/{slug}/details', get_forum, name = 'forum_details')
    app.router.add_post('/api/forum/{slug}/create', create_thread, name = 'new_thread')
    app.router.add_post('/api/thread/{slug_or_id}/create', create_post, name = 'new_post')
    app.router.add_get('/api/thread/{slug_or_id}/details', get_thread, name = 'thread_details')
    app.router.add_get('/api/forum/{slug}/threads', get_forum_threads, name = 'forum_threads')
    app.router.add_post('/api/service/clear', clear, name = 'clear')
    app.router.add_get('/api/service/status', get_status, name = 'status')
    app.router.add_post('/api/thread/{slug_or_id}/vote', thread_vote, name = 'new_vote')
    app.router.add_post('/api/thread/{slug_or_id}/details', update_thread, name = 'thread_update')
    app.router.add_get('/api/thread/{slug_or_id}/posts', get_thread_posts, name = 'thread_posts')
