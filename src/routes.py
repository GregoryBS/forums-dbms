from .views import *

def setup_routes(app):
    app.router.add_get('/', index)
    app.router.add_post('/api/user/{nick}/create', signup, name = 'signup')
    app.router.add_get('/api/user/{nick}/profile', get_profile, name = 'personal')
    app.router.add_post('/api/user/{nick}/profile', update_profile, name = 'personal_edit')