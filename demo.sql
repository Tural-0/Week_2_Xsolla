-- select * from items
-- where id = 1;

-- insert into items (name, description, price, stock)
-- values ('Coffee Mug', 'Xsolla branded coffee mug', 999, 5);

-- explain select * from items
-- order by id desc


-- update items set stock=4
-- where id=9;

-- delete from items
-- where id = 9;

-- select i.description, oi.quantity from order_items as oi
-- left join items as i
-- on oi.item_id=i.id
