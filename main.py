import asyncio
import asyncpg

async def main():
    conn = await asyncpg.connect('postgresql://anna:yoh@localhost:5432/forums')

    print("all is ok")
    await conn.close()

asyncio.get_event_loop().run_until_complete(main())
