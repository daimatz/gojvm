import java.util.Iterator;

public class CustomIterator implements Iterable<String> {
    private String[] data;

    public CustomIterator(String[] data) {
        this.data = data;
    }

    public Iterator<String> iterator() {
        return new MyIter();
    }

    private class MyIter implements Iterator<String> {
        private int index = 0;
        public boolean hasNext() { return index < data.length; }
        public String next() { return data[index++]; }
    }

    public static void main(String[] args) {
        CustomIterator ci = new CustomIterator(new String[]{"X", "Y", "Z"});
        for (String s : ci) {
            System.out.println(s);
        }
    }
}
